package domain_test

import (
	"testing"
	"time"

	"github.com/kstu/tabel/internal/domain"
)

func TestDisplayName(t *testing.T) {
	got := domain.DisplayName("Жумалиев", "Арслан", "")
	if got != "Жумалиев Арслан" {
		t.Fatalf("got %q", got)
	}
}

func TestIsWeekend(t *testing.T) {
	sat := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC) // Saturday
	code, ok := domain.IsWeekend(sat)
	if !ok || code != "сб" {
		t.Fatalf("saturday: %v %s", ok, code)
	}
}
