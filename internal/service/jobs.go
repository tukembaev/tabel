package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kstu/tabel/internal/store"
)

func (s *Service) AutoCloseStaffDays(ctx context.Context) (int, error) {
	now := s.now()
	wd := s.workDate(now)
	cutoff := s.cutoffFor(wd)
	if now.Before(cutoff) {
		return 0, nil
	}
	open, err := s.q.ListOpenDaySessionsForDate(ctx, store.ListOpenDaySessionsForDateParams{
		UniversityID: s.cfg.UniversityID,
		WorkDate:     wd,
	})
	if err != nil {
		return 0, err
	}
	n := 0
	closeAt := cutoff
	for _, sess := range open {
		dur := s.durationSeconds(sess.OpenedAt, closeAt)
		lunch := s.lunchSeconds(sess.OpenedAt, closeAt)
		if _, err := s.q.CloseDaySession(ctx, store.CloseDaySessionParams{
			ID:              sess.ID,
			ClosedAt:        store.Timestamptz(closeAt.UTC()),
			CloseReason:     store.Text("auto_cutoff"),
			DurationSeconds: store.Int4(dur),
			LunchSeconds:    lunch,
		}); err != nil {
			return n, fmt.Errorf("close %s: %w", sess.ID, err)
		}
		n++
	}
	return n, nil
}

func (s *Service) ArchiveOld(ctx context.Context) (sessions int, exceptions int, err error) {
	cutoff, err := s.archiveCutoff()
	if err != nil {
		return 0, 0, err
	}
	const batch = 500
	for {
		ids, err := s.q.ListSessionIDsBefore(ctx, store.ListSessionIDsBeforeParams{
			UniversityID: s.cfg.UniversityID,
			WorkDate:     cutoff,
			Limit:        batch,
		})
		if err != nil {
			return sessions, exceptions, err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			if err := s.q.MoveSessionToArchive(ctx, id); err != nil {
				return sessions, exceptions, err
			}
			if err := s.q.DeleteSession(ctx, id); err != nil {
				return sessions, exceptions, err
			}
			sessions++
		}
	}
	for {
		ids, err := s.q.ListExceptionIDsBefore(ctx, store.ListExceptionIDsBeforeParams{
			UniversityID: s.cfg.UniversityID,
			WorkDate:     cutoff,
			Limit:        batch,
		})
		if err != nil {
			return sessions, exceptions, err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			if err := s.q.MoveExceptionToArchive(ctx, id); err != nil {
				return sessions, exceptions, err
			}
			if err := s.q.DeleteException(ctx, id); err != nil {
				return sessions, exceptions, err
			}
			exceptions++
		}
	}
	_, err = s.q.InsertArchiveRun(ctx, store.InsertArchiveRunParams{
		UniversityID:    s.cfg.UniversityID,
		CutoffDate:      cutoff,
		SessionsMoved:   int32(sessions),
		ExceptionsMoved: int32(exceptions),
	})
	return sessions, exceptions, err
}

func (s *Service) archiveCutoff() (time.Time, error) {
	prev, err := time.ParseInLocation("2006-01-02", s.cfg.PrevSemesterStart, s.loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("PREV_SEMESTER_START: %w", err)
	}
	return prev.AddDate(0, 0, -s.cfg.ArchiveGraceDays), nil
}
