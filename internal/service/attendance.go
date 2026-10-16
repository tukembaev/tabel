package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kstu/tabel/internal/clock"
	"github.com/kstu/tabel/internal/config"
	"github.com/kstu/tabel/internal/domain"
	"github.com/kstu/tabel/internal/store"
)

type ScanAction string

const (
	ScanActionCheckIn          ScanAction = "check_in"
	ScanActionCheckOut         ScanAction = "check_out"
	ScanActionDuplicateIgnored ScanAction = "duplicate_ignored"
	ScanActionAlreadyClosed    ScanAction = "already_closed"
)

type LatestSession struct {
	Status      string
	OpenedAt    time.Time
	ClosedAt    time.Time
	CloseReason string
}

type Service struct {
	cfg     config.Config
	db      *pgxpool.Pool
	q       *store.Queries
	clk     clock.Clock
	loc     *time.Location
	startH  int
	startM  int
	cutH    int
	cutM    int
	lunchH  int
	lunchM  int
	lunchEH int
	lunchEM int
}

func New(cfg config.Config, db *pgxpool.Pool, clk clock.Clock) (*Service, error) {
	loc, err := cfg.Location()
	if err != nil {
		return nil, err
	}
	startH, startM, err := cfg.StaffStartClock()
	if err != nil {
		return nil, err
	}
	cutH, cutM, err := cfg.StaffCutoffClock()
	if err != nil {
		return nil, err
	}
	lunchH, lunchM, err := cfg.LunchStartClock()
	if err != nil {
		return nil, err
	}
	lunchEH, lunchEM, err := cfg.LunchEndClock()
	if err != nil {
		return nil, err
	}
	if clk == nil {
		clk = clock.Real{}
	}
	return &Service{
		cfg: cfg, db: db, q: store.New(db), clk: clk, loc: loc,
		startH: startH, startM: startM, cutH: cutH, cutM: cutM,
		lunchH: lunchH, lunchM: lunchM, lunchEH: lunchEH, lunchEM: lunchEM,
	}, nil
}

func (s *Service) now() time.Time { return s.clk.Now().In(s.loc) }

func (s *Service) workDate(t time.Time) time.Time {
	y, m, d := t.In(s.loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Service) clockOn(workDate time.Time, hour, min int) time.Time {
	y, m, d := workDate.In(time.UTC).Date()
	return time.Date(y, m, d, hour, min, 0, 0, s.loc)
}

func (s *Service) cutoffFor(workDate time.Time) time.Time {
	return s.clockOn(workDate, s.cutH, s.cutM)
}

func (s *Service) lunchSeconds(openedAt, closedAt time.Time) int32 {
	opened := openedAt.In(s.loc)
	closed := closedAt.In(s.loc)
	y, m, d := opened.Date()
	start := time.Date(y, m, d, s.lunchH, s.lunchM, 0, 0, s.loc)
	end := time.Date(y, m, d, s.lunchEH, s.lunchEM, 0, 0, s.loc)
	return int32(domain.LunchOverlapSeconds(opened, closed, start, end))
}

func (s *Service) durationSeconds(openedAt, closedAt time.Time) int32 {
	d := closedAt.Sub(openedAt)
	if d < 0 {
		return 0
	}
	return int32(d / time.Second)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func canManageDepartment(snap store.UserSnapshot) bool {
	return snap.IsDepartmentHead || snap.CanEditTimesheet
}

func sameDepartment(a, b store.UserSnapshot) bool {
	aid := store.TextValue(a.DepartmentID)
	bid := store.TextValue(b.DepartmentID)
	return aid != "" && aid == bid
}

type SyncInput struct {
	Username         string `json:"username"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	MiddleName       string `json:"middle_name"`
	HasEmployee      bool   `json:"has_employee"`
	IsDepartmentHead bool   `json:"is_department_head"`
	CanEditTimesheet bool   `json:"can_edit_timesheet"`
	DepartmentID     string `json:"department_id"`
	DepartmentName   string `json:"department_name"`
}

func (s *Service) SyncMe(ctx context.Context, userID uuid.UUID, in SyncInput) (store.UserSnapshot, error) {
	return s.q.UpsertUserSnapshot(ctx, store.UpsertUserSnapshotParams{
		UserID:           userID,
		UniversityID:     s.cfg.UniversityID,
		Username:         in.Username,
		FirstName:        in.FirstName,
		LastName:         in.LastName,
		MiddleName:       in.MiddleName,
		HasEmployee:      in.HasEmployee,
		IsDepartmentHead: in.IsDepartmentHead,
		CanEditTimesheet: in.CanEditTimesheet || in.IsDepartmentHead,
		DepartmentID:     store.Text(in.DepartmentID),
		DepartmentName:   in.DepartmentName,
	})
}

func (s *Service) requireSnapshot(ctx context.Context, userID uuid.UUID) (store.UserSnapshot, error) {
	snap, err := s.q.GetUserSnapshot(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.UserSnapshot{}, domain.ErrSnapshotRequired
	}
	return snap, err
}

func (s *Service) requireEmployee(snap store.UserSnapshot) error {
	if !snap.HasEmployee {
		return domain.ErrForbidden
	}
	return nil
}

type ScanInput struct {
	QRURL string `json:"qr_url"`
}

type ScanResult struct {
	Action  ScanAction              `json:"action"`
	Session store.AttendanceSession `json:"session"`
}

// DecideScanAction: one in/out cycle per day. Lunch is never a extra cycle.
// A late scan after auto_cutoff overrides 18:30 with the real exit time.
func DecideScanAction(latest *LatestSession, now, cutoff time.Time, debounce time.Duration) (ScanAction, error) {
	if latest == nil {
		if !now.Before(cutoff) {
			return "", domain.ErrAfterCutoff
		}
		return ScanActionCheckIn, nil
	}
	switch latest.Status {
	case "open":
		if now.Sub(latest.OpenedAt) < debounce {
			return ScanActionDuplicateIgnored, nil
		}
		return ScanActionCheckOut, nil
	case "closed":
		closedAt := latest.OpenedAt
		if !latest.ClosedAt.IsZero() {
			closedAt = latest.ClosedAt
		}
		if now.Sub(closedAt) < debounce {
			return ScanActionDuplicateIgnored, nil
		}
		if latest.CloseReason == "auto_cutoff" {
			return ScanActionCheckOut, nil
		}
		return ScanActionAlreadyClosed, nil
	default:
		return "", fmt.Errorf("%w: unknown session status", domain.ErrValidation)
	}
}

func (s *Service) Scan(ctx context.Context, userID uuid.UUID, in ScanInput) (ScanResult, error) {
	loc, err := domain.ParseQRURL(in.QRURL)
	if err != nil {
		return ScanResult{}, err
	}
	if loc.UniversityID != s.cfg.UniversityID {
		return ScanResult{}, fmt.Errorf("%w: university mismatch", domain.ErrInvalidQR)
	}

	snap, err := s.requireSnapshot(ctx, userID)
	if err != nil {
		return ScanResult{}, err
	}
	if err := s.requireEmployee(snap); err != nil {
		return ScanResult{}, err
	}

	now := s.now()
	wd := s.workDate(now)
	cutoff := s.cutoffFor(wd)

	var latestPtr *LatestSession
	var latestSess store.AttendanceSession
	latestSess, err = s.q.GetLatestDaySession(ctx, store.GetLatestDaySessionParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       userID,
		WorkDate:     wd,
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		latestPtr = nil
	case err != nil:
		return ScanResult{}, err
	default:
		ls := LatestSession{Status: latestSess.Status, OpenedAt: latestSess.OpenedAt, CloseReason: store.TextValue(latestSess.CloseReason)}
		if latestSess.ClosedAt.Valid {
			ls.ClosedAt = latestSess.ClosedAt.Time
		}
		latestPtr = &ls
	}

	action, err := DecideScanAction(latestPtr, now, cutoff, s.cfg.DuplicateScanWindow)
	if err != nil {
		return ScanResult{}, err
	}

	switch action {
	case ScanActionDuplicateIgnored, ScanActionAlreadyClosed:
		return ScanResult{Action: action, Session: latestSess}, nil
	case ScanActionCheckOut:
		dur := s.durationSeconds(latestSess.OpenedAt, now)
		lunch := s.lunchSeconds(latestSess.OpenedAt, now)
		closedAt := store.Timestamptz(now.UTC())
		var sess store.AttendanceSession
		if latestSess.Status == "closed" {
			sess, err = s.q.OverrideAutoClose(ctx, store.OverrideAutoCloseParams{
				ID:              latestSess.ID,
				ClosedAt:        closedAt,
				ExitBuilding:    store.Text(loc.BuildingID),
				ExitRoom:        store.Text(loc.RoomNumber),
				DurationSeconds: store.Int4(dur),
				LunchSeconds:    lunch,
			})
		} else {
			sess, err = s.q.CloseDaySession(ctx, store.CloseDaySessionParams{
				ID:              latestSess.ID,
				ClosedAt:        closedAt,
				CloseReason:     store.Text("manual"),
				ExitBuilding:    store.Text(loc.BuildingID),
				ExitRoom:        store.Text(loc.RoomNumber),
				DurationSeconds: store.Int4(dur),
				LunchSeconds:    lunch,
			})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return ScanResult{}, domain.ErrNoOpenSession
		}
		if err != nil {
			return ScanResult{}, err
		}
		return ScanResult{Action: action, Session: sess}, nil
	case ScanActionCheckIn:
		if _, err := s.q.GetApprovedException(ctx, store.GetApprovedExceptionParams{
			UniversityID: s.cfg.UniversityID,
			UserID:       userID,
			WorkDate:     wd,
		}); err == nil {
			return ScanResult{}, fmt.Errorf("%w: approved absence for today", domain.ErrConflict)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return ScanResult{}, err
		}
		sess, err := s.q.CreateDaySession(ctx, store.CreateDaySessionParams{
			UniversityID:  s.cfg.UniversityID,
			UserID:        userID,
			WorkDate:      wd,
			DepartmentID:  snap.DepartmentID,
			EntryBuilding: loc.BuildingID,
			EntryRoom:     loc.RoomNumber,
			OpenedAt:      now.UTC(),
			DisplayName:   domain.DisplayName(snap.LastName, snap.FirstName, snap.MiddleName),
		})
		if isUniqueViolation(err) {
			existing, getErr := s.q.GetLatestDaySession(ctx, store.GetLatestDaySessionParams{
				UniversityID: s.cfg.UniversityID,
				UserID:       userID,
				WorkDate:     wd,
			})
			if getErr == nil {
				return ScanResult{Action: ScanActionDuplicateIgnored, Session: existing}, nil
			}
			return ScanResult{}, fmt.Errorf("%w: open session already exists", domain.ErrConflict)
		}
		if err != nil {
			return ScanResult{}, err
		}
		return ScanResult{Action: action, Session: sess}, nil
	default:
		return ScanResult{}, fmt.Errorf("%w: unknown action", domain.ErrValidation)
	}
}
