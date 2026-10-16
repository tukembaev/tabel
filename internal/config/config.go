package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTPAddr            string        `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL         string        `env:"DATABASE_URL" envDefault:"postgres://tabel:tabel@localhost:5432/tabel?sslmode=disable"`
	RedisURL            string        `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
	JWTSecret           string        `env:"JWT_SECRET,required"`
	UniversityID        string        `env:"UNIVERSITY_ID" envDefault:"kstu"`
	Timezone            string        `env:"TIMEZONE" envDefault:"Asia/Bishkek"`
	StaffStartHM        string        `env:"STAFF_START" envDefault:"08:00"`
	StaffCutoffHM       string        `env:"STAFF_CUTOFF" envDefault:"18:30"`
	LunchStartHM        string        `env:"LUNCH_START" envDefault:"12:00"`
	LunchEndHM          string        `env:"LUNCH_END" envDefault:"13:00"`
	DuplicateScanWindow time.Duration `env:"DUPLICATE_SCAN_WINDOW" envDefault:"2m"`
	// Academic year / archive
	SemesterStart     string `env:"SEMESTER_START" envDefault:"2025-09-01"`
	PrevSemesterStart string `env:"PREV_SEMESTER_START" envDefault:"2025-02-01"`
	ArchiveGraceDays  int    `env:"ARCHIVE_GRACE_DAYS" envDefault:"30"`
	// Local / Swagger testing: Basic Auth user:user maps to this UUID and can set access_token cookie.
	BasicAuthUser     string `env:"BASIC_AUTH_USER" envDefault:"user"`
	BasicAuthPassword string `env:"BASIC_AUTH_PASSWORD" envDefault:"user"`
	DevUserID         string `env:"DEV_USER_ID" envDefault:"15b7fbe4-95d0-4319-a6c5-fa1ea13391ed"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

func (c Config) Location() (*time.Location, error) {
	return time.LoadLocation(c.Timezone)
}

func (c Config) StaffCutoffClock() (hour, min int, err error) {
	return parseClock("STAFF_CUTOFF", c.StaffCutoffHM)
}

func (c Config) StaffStartClock() (hour, min int, err error) {
	return parseClock("STAFF_START", c.StaffStartHM)
}

func (c Config) LunchStartClock() (hour, min int, err error) {
	return parseClock("LUNCH_START", c.LunchStartHM)
}

func (c Config) LunchEndClock() (hour, min int, err error) {
	return parseClock("LUNCH_END", c.LunchEndHM)
}

func parseClock(name, hm string) (hour, min int, err error) {
	t, err := time.Parse("15:04", hm)
	if err != nil {
		return 0, 0, fmt.Errorf("%s: %w", name, err)
	}
	return t.Hour(), t.Minute(), nil
}
