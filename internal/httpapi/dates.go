package httpapi

import (
	"fmt"
	"time"
)

// parseAPIDate accepts YYYY-MM-DD or RFC3339.
func parseAPIDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("date must be YYYY-MM-DD or RFC3339")
}

func parseOptionalRFC3339(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("datetime must be RFC3339")
	}
	return &t, nil
}
