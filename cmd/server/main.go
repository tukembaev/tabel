package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kstu/tabel/internal/auth"
	"github.com/kstu/tabel/internal/config"
	"github.com/kstu/tabel/internal/httpapi"
	"github.com/kstu/tabel/internal/httpapi/middleware"
	"github.com/kstu/tabel/internal/service"
)

// Set via: go build -ldflags "-X main.version=..."
var version = "dev"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Error("db ping", "err", err)
		os.Exit(1)
	}

	svc, err := service.New(cfg, pool, nil)
	if err != nil {
		log.Error("service", "err", err)
		os.Exit(1)
	}
	devUID, err := uuid.Parse(cfg.DevUserID)
	if err != nil {
		log.Error("DEV_USER_ID", "err", err)
		os.Exit(1)
	}
	authCfg := middleware.AuthConfig{
		BasicUser:     cfg.BasicAuthUser,
		BasicPassword: cfg.BasicAuthPassword,
		BasicUserID:   devUID,
	}
	api := httpapi.New(svc, auth.NewVerifier(cfg.JWTSecret), authCfg, cfg.BasicAuthUser, cfg.BasicAuthPassword)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("http listening", "addr", cfg.HTTPAddr, "version", version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
