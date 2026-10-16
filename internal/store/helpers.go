package store

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func Text(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func TextPtr(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func TextValue(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func UUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func NullUUID() pgtype.UUID {
	return pgtype.UUID{}
}

func Timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func NullTimestamptz() pgtype.Timestamptz {
	return pgtype.Timestamptz{}
}

func Int4(n int32) pgtype.Int4 {
	return pgtype.Int4{Int32: n, Valid: true}
}

func TimeValue(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
