package domain_test

import (
	"testing"
	"time"

	"github.com/kstu/tabel/internal/domain"
)

func TestLunchOverlapSeconds(t *testing.T) {
	loc := time.UTC
	lunchStart := time.Date(2026, 8, 7, 12, 0, 0, 0, loc)
	lunchEnd := time.Date(2026, 8, 7, 13, 0, 0, 0, loc)

	full := domain.LunchOverlapSeconds(
		time.Date(2026, 8, 7, 9, 0, 0, 0, loc),
		time.Date(2026, 8, 7, 18, 0, 0, 0, loc),
		lunchStart, lunchEnd,
	)
	if full != 3600 {
		t.Fatalf("full day lunch: %d", full)
	}

	before := domain.LunchOverlapSeconds(
		time.Date(2026, 8, 7, 8, 0, 0, 0, loc),
		time.Date(2026, 8, 7, 12, 0, 0, 0, loc),
		lunchStart, lunchEnd,
	)
	if before != 0 {
		t.Fatalf("left at lunch start: %d", before)
	}

	late := domain.LunchOverlapSeconds(
		time.Date(2026, 8, 7, 9, 0, 0, 0, loc),
		time.Date(2026, 8, 7, 21, 0, 0, 0, loc),
		lunchStart, lunchEnd,
	)
	if late != 3600 {
		t.Fatalf("stayed until 21: %d", late)
	}
}
