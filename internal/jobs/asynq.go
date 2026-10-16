package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/kstu/tabel/internal/service"
)

const (
	TypeCloseStaffDays = "tabel:close_staff_days"
	TypeArchive        = "tabel:archive"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeCloseStaffDays, h.handleCloseStaff)
	mux.HandleFunc(TypeArchive, h.handleArchive)
}

func (h *Handler) handleCloseStaff(ctx context.Context, _ *asynq.Task) error {
	n, err := h.svc.AutoCloseStaffDays(ctx)
	if err != nil {
		return err
	}
	h.log.Info("auto-closed staff days", "count", n)
	return nil
}

func (h *Handler) handleArchive(ctx context.Context, _ *asynq.Task) error {
	s, e, err := h.svc.ArchiveOld(ctx)
	if err != nil {
		return err
	}
	h.log.Info("archive run", "sessions", s, "exceptions", e)
	return nil
}

func EnqueuePeriodic(sched *asynq.Scheduler) error {
	if _, err := sched.Register("@every 1m", asynq.NewTask(TypeCloseStaffDays, nil)); err != nil {
		return fmt.Errorf("register close staff: %w", err)
	}
	if _, err := sched.Register("@daily", asynq.NewTask(TypeArchive, nil)); err != nil {
		return fmt.Errorf("register archive: %w", err)
	}
	return nil
}

func ParseRedisOpt(redisURL string) (asynq.RedisClientOpt, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return asynq.RedisClientOpt{Addr: "localhost:6379"}, err
	}
	if c, ok := opt.(asynq.RedisClientOpt); ok {
		return c, nil
	}
	return asynq.RedisClientOpt{Addr: "localhost:6379"}, fmt.Errorf("unsupported redis opt type")
}
