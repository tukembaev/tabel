package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kstu/tabel/internal/domain"
	"github.com/kstu/tabel/internal/store"
)

type EditDayInput struct {
	UserID   uuid.UUID
	WorkDate time.Time
	Kind     string // attendance | absence | clear
	OpenedAt *time.Time
	ClosedAt *time.Time
	Code     string
	Note     string
}

type EditDayResult struct {
	Kind      string                   `json:"kind"`
	Session   *store.AttendanceSession `json:"session,omitempty"`
	Exception *store.DayException      `json:"exception,omitempty"`
}

func (s *Service) requireDeptManager(ctx context.Context, editorID uuid.UUID) (store.UserSnapshot, error) {
	editor, err := s.requireSnapshot(ctx, editorID)
	if err != nil {
		return store.UserSnapshot{}, err
	}
	if !canManageDepartment(editor) {
		return store.UserSnapshot{}, domain.ErrForbidden
	}
	if store.TextValue(editor.DepartmentID) == "" {
		return store.UserSnapshot{}, domain.ErrForbidden
	}
	return editor, nil
}

func (s *Service) requireDeptManagerOf(ctx context.Context, editorID uuid.UUID, departmentID string) (store.UserSnapshot, error) {
	editor, err := s.requireDeptManager(ctx, editorID)
	if err != nil {
		return store.UserSnapshot{}, err
	}
	if departmentID != "" && store.TextValue(editor.DepartmentID) != departmentID {
		return store.UserSnapshot{}, domain.ErrForbidden
	}
	return editor, nil
}

func (s *Service) EditDepartmentDay(ctx context.Context, editorID uuid.UUID, in EditDayInput) (EditDayResult, error) {
	editor, err := s.requireDeptManager(ctx, editorID)
	if err != nil {
		return EditDayResult{}, err
	}
	target, err := s.requireSnapshot(ctx, in.UserID)
	if err != nil {
		return EditDayResult{}, err
	}
	if !sameDepartment(editor, target) {
		return EditDayResult{}, domain.ErrForbidden
	}
	wd := s.workDate(in.WorkDate)
	payload, _ := json.Marshal(map[string]any{
		"kind":      in.Kind,
		"opened_at": in.OpenedAt,
		"closed_at": in.ClosedAt,
		"code":      in.Code,
		"note":      in.Note,
	})

	switch in.Kind {
	case "attendance":
		res, err := s.headSetAttendance(ctx, editor, target, wd, in)
		if err != nil {
			return EditDayResult{}, err
		}
		_, _ = s.q.InsertTimesheetEdit(ctx, store.InsertTimesheetEditParams{
			UniversityID: s.cfg.UniversityID,
			DepartmentID: target.DepartmentID,
			EditorID:     editorID,
			UserID:       in.UserID,
			WorkDate:     wd,
			Action:       "set_attendance",
			Note:         in.Note,
			Payload:      payload,
		})
		return res, nil
	case "absence":
		res, err := s.headSetAbsenceDay(ctx, editor, target, wd, in)
		if err != nil {
			return EditDayResult{}, err
		}
		_, _ = s.q.InsertTimesheetEdit(ctx, store.InsertTimesheetEditParams{
			UniversityID: s.cfg.UniversityID,
			DepartmentID: target.DepartmentID,
			EditorID:     editorID,
			UserID:       in.UserID,
			WorkDate:     wd,
			Action:       "set_absence",
			Note:         in.Note,
			Payload:      payload,
		})
		return res, nil
	case "clear":
		if err := s.q.DeleteSessionsForUserDate(ctx, store.DeleteSessionsForUserDateParams{
			UniversityID: s.cfg.UniversityID,
			UserID:       in.UserID,
			WorkDate:     wd,
		}); err != nil {
			return EditDayResult{}, err
		}
		if err := s.q.DeleteExceptionsForUserDate(ctx, store.DeleteExceptionsForUserDateParams{
			UniversityID: s.cfg.UniversityID,
			UserID:       in.UserID,
			WorkDate:     wd,
		}); err != nil {
			return EditDayResult{}, err
		}
		_, _ = s.q.InsertTimesheetEdit(ctx, store.InsertTimesheetEditParams{
			UniversityID: s.cfg.UniversityID,
			DepartmentID: target.DepartmentID,
			EditorID:     editorID,
			UserID:       in.UserID,
			WorkDate:     wd,
			Action:       "clear",
			Note:         in.Note,
			Payload:      payload,
		})
		return EditDayResult{Kind: "clear"}, nil
	default:
		return EditDayResult{}, fmt.Errorf("%w: kind must be attendance, absence or clear", domain.ErrValidation)
	}
}

func (s *Service) headSetAttendance(ctx context.Context, editor, target store.UserSnapshot, wd time.Time, in EditDayInput) (EditDayResult, error) {
	opened := s.clockOn(wd, s.startH, s.startM)
	closed := s.clockOn(wd, s.cutH, s.cutM)
	if in.OpenedAt != nil {
		opened = in.OpenedAt.In(s.loc)
	}
	if in.ClosedAt != nil {
		closed = in.ClosedAt.In(s.loc)
	}
	if !closed.After(opened) {
		return EditDayResult{}, fmt.Errorf("%w: closed_at must be after opened_at", domain.ErrValidation)
	}
	if err := s.q.DeleteExceptionsForUserDate(ctx, store.DeleteExceptionsForUserDateParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       target.UserID,
		WorkDate:     wd,
	}); err != nil {
		return EditDayResult{}, err
	}
	dur := s.durationSeconds(opened, closed)
	lunch := s.lunchSeconds(opened, closed)
	sess, err := s.q.UpsertHeadDaySession(ctx, store.UpsertHeadDaySessionParams{
		UniversityID:    s.cfg.UniversityID,
		UserID:          target.UserID,
		WorkDate:        wd,
		DepartmentID:    target.DepartmentID,
		OpenedAt:        opened.UTC(),
		ClosedAt:        store.Timestamptz(closed.UTC()),
		DurationSeconds: store.Int4(dur),
		LunchSeconds:    lunch,
		DisplayName:     domain.DisplayName(target.LastName, target.FirstName, target.MiddleName),
		EditedBy:        store.UUID(editor.UserID),
		Note:            in.Note,
	})
	if err != nil {
		return EditDayResult{}, err
	}
	return EditDayResult{Kind: "attendance", Session: &sess}, nil
}

func (s *Service) headSetAbsenceDay(ctx context.Context, editor, target store.UserSnapshot, wd time.Time, in EditDayInput) (EditDayResult, error) {
	if _, ok := domain.AllCodes[in.Code]; !ok {
		return EditDayResult{}, domain.ErrInvalidCode
	}
	if err := s.q.DeleteSessionsForUserDate(ctx, store.DeleteSessionsForUserDateParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       target.UserID,
		WorkDate:     wd,
	}); err != nil {
		return EditDayResult{}, err
	}
	if err := s.q.DeleteExceptionsForUserDate(ctx, store.DeleteExceptionsForUserDateParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       target.UserID,
		WorkDate:     wd,
	}); err != nil {
		return EditDayResult{}, err
	}
	ex, err := s.q.CreateDayException(ctx, store.CreateDayExceptionParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       target.UserID,
		DepartmentID: target.DepartmentID,
		WorkDate:     wd,
		Code:         in.Code,
		Status:       "approved",
		Source:       "head",
		Note:         in.Note,
		RequestedBy:  store.UUID(editor.UserID),
		DecidedBy:    store.UUID(editor.UserID),
		DecidedAt:    store.Timestamptz(s.now().UTC()),
		DisplayName:  domain.DisplayName(target.LastName, target.FirstName, target.MiddleName),
	})
	if err != nil {
		return EditDayResult{}, err
	}
	return EditDayResult{Kind: "absence", Exception: &ex}, nil
}

func (s *Service) ListDepartmentEmployees(ctx context.Context, editorID uuid.UUID, departmentID string) ([]store.UserSnapshot, error) {
	editor, err := s.requireDeptManagerOf(ctx, editorID, departmentID)
	if err != nil {
		return nil, err
	}
	dept := store.TextValue(editor.DepartmentID)
	return s.q.ListEmployeesByDepartment(ctx, store.ListEmployeesByDepartmentParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: store.Text(dept),
	})
}

func (s *Service) ListDepartmentEdits(ctx context.Context, editorID uuid.UUID, departmentID string, from, to time.Time) ([]store.TimesheetEdit, error) {
	editor, err := s.requireDeptManagerOf(ctx, editorID, departmentID)
	if err != nil {
		return nil, err
	}
	return s.q.ListTimesheetEditsByDept(ctx, store.ListTimesheetEditsByDeptParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: editor.DepartmentID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
}
