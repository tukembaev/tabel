package service

import (
	"errors"
	"testing"
	"time"

	"github.com/kstu/tabel/internal/domain"
)

func TestDecideScanAction(t *testing.T) {
	loc := time.UTC
	cutoff := time.Date(2026, 8, 7, 18, 30, 0, 0, loc)
	debounce := 2 * time.Minute

	t.Run("first scan is check_in", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		got, err := DecideScanAction(nil, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionCheckIn {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("duplicate within window is ignored", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 8, 1, 0, 0, loc)
		opened := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{Status: "open", OpenedAt: opened}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionDuplicateIgnored {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("scan after window is check_out", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 9, 0, 0, 0, loc)
		opened := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{Status: "open", OpenedAt: opened}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionCheckOut {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("third scan after manual checkout is already_closed", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 13, 0, 0, 0, loc)
		opened := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		closed := time.Date(2026, 8, 7, 12, 0, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{
			Status:      "closed",
			OpenedAt:    opened,
			ClosedAt:    closed,
			CloseReason: "manual",
		}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionAlreadyClosed {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("new check_in after cutoff is rejected", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 19, 0, 0, 0, loc)
		_, err := DecideScanAction(nil, now, cutoff, debounce)
		if !errors.Is(err, domain.ErrAfterCutoff) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("checkout after cutoff is allowed", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 19, 0, 0, 0, loc)
		opened := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{Status: "open", OpenedAt: opened}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionCheckOut {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("late scan overrides auto_cutoff", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 21, 0, 0, 0, loc)
		opened := time.Date(2026, 8, 7, 9, 0, 0, 0, loc)
		closed := time.Date(2026, 8, 7, 18, 30, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{
			Status:      "closed",
			OpenedAt:    opened,
			ClosedAt:    closed,
			CloseReason: "auto_cutoff",
		}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionCheckOut {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("manual close after cutoff stays closed", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 19, 0, 0, 0, loc)
		opened := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		closed := time.Date(2026, 8, 7, 18, 0, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{
			Status:      "closed",
			OpenedAt:    opened,
			ClosedAt:    closed,
			CloseReason: "manual",
		}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionAlreadyClosed {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("duplicate right after checkout is ignored", func(t *testing.T) {
		now := time.Date(2026, 8, 7, 18, 0, 30, 0, loc)
		opened := time.Date(2026, 8, 7, 8, 0, 0, 0, loc)
		closed := time.Date(2026, 8, 7, 18, 0, 0, 0, loc)
		got, err := DecideScanAction(&LatestSession{
			Status:      "closed",
			OpenedAt:    opened,
			ClosedAt:    closed,
			CloseReason: "manual",
		}, now, cutoff, debounce)
		if err != nil {
			t.Fatal(err)
		}
		if got != ScanActionDuplicateIgnored {
			t.Fatalf("got %s", got)
		}
	})
}
