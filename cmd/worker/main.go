package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kstu/tabel/internal/config"
	"github.com/kstu/tabel/internal/jobs"
	"github.com/kstu/tabel/internal/service"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	svc, err := service.New(cfg, pool, nil)
	if err != nil {
		log.Error("service", "err", err)
		os.Exit(1)
	}

	redisOpt, err := jobs.ParseRedisOpt(cfg.RedisURL)
	if err != nil {
		log.Warn("redis parse, using fallback addr", "err", err)
		redisOpt = asynq.RedisClientOpt{Addr: "localhost:6379"}
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{Concurrency: 4})
	mux := asynq.NewServeMux()
	jobs.NewHandler(svc, log).Register(mux)

	sched := asynq.NewScheduler(redisOpt, nil)
	if err := jobs.EnqueuePeriodic(sched); err != nil {
		log.Error("scheduler register", "err", err)
		os.Exit(1)
	}

	go func() {
		log.Info("asynq scheduler starting")
		if err := sched.Run(); err != nil {
			log.Error("scheduler", "err", err)
			os.Exit(1)
		}
	}()

	go func() {
		log.Info("asynq worker starting")
		if err := srv.Run(mux); err != nil {
			log.Error("worker", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	sched.Shutdown()
	srv.Shutdown()
}
