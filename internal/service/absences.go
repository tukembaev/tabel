package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kstu/tabel/internal/domain"
	"github.com/kstu/tabel/internal/store"
)

type AbsenceRequestInput struct {
	WorkDate time.Time `json:"work_date"`
	Code     string    `json:"code"`
	Note     string    `json:"note"`
}

func (s *Service) RequestAbsence(ctx context.Context, userID uuid.UUID, in AbsenceRequestInput) (store.DayException, error) {
	snap, err := s.requireSnapshot(ctx, userID)
	if err != nil {
		return store.DayException{}, err
	}
	if !snap.HasEmployee {
		return store.DayException{}, domain.ErrForbidden
	}
	if _, ok := domain.HeadOnlyCodes[in.Code]; ok {
		return store.DayException{}, domain.ErrHeadOnlyCode
	}
	if _, ok := domain.AllCodes[in.Code]; !ok {
		if _, ok := domain.EmployeeRequestCodes[in.Code]; !ok {
			return store.DayException{}, domain.ErrInvalidCode
		}
	}
	wd := s.workDate(in.WorkDate)
	if err := s.ensureNoCheckIn(ctx, userID, wd); err != nil {
		return store.DayException{}, err
	}
	ex, err := s.q.CreateDayException(ctx, store.CreateDayExceptionParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       userID,
		DepartmentID: snap.DepartmentID,
		WorkDate:     wd,
		Code:         in.Code,
		Status:       "pending",
		Source:       "employee_request",
		Note:         in.Note,
		RequestedBy:  store.UUID(userID),
		DecidedBy:    store.NullUUID(),
		DecidedAt:    store.NullTimestamptz(),
		DisplayName:  domain.DisplayName(snap.LastName, snap.FirstName, snap.MiddleName),
	})
	if isUniqueViolation(err) {
		return store.DayException{}, fmt.Errorf("%w: exception already exists", domain.ErrConflict)
	}
	return ex, err
}

type HeadSetAbsenceInput struct {
	UserID   uuid.UUID `json:"user_id"`
	WorkDate time.Time `json:"work_date"`
	Code     string    `json:"code"`
	Note     string    `json:"note"`
}

func (s *Service) HeadSetAbsence(ctx context.Context, headID uuid.UUID, in HeadSetAbsenceInput) (store.DayException, error) {
	head, err := s.requireSnapshot(ctx, headID)
	if err != nil {
		return store.DayException{}, err
	}
	if !canManageDepartment(head) {
		return store.DayException{}, domain.ErrForbidden
	}
	if _, ok := domain.AllCodes[in.Code]; !ok {
		return store.DayException{}, domain.ErrInvalidCode
	}
	target, err := s.requireSnapshot(ctx, in.UserID)
	if err != nil {
		return store.DayException{}, err
	}
	if store.TextValue(head.DepartmentID) == "" || store.TextValue(head.DepartmentID) != store.TextValue(target.DepartmentID) {
		return store.DayException{}, domain.ErrForbidden
	}
	wd := s.workDate(in.WorkDate)
	if err := s.ensureNoCheckIn(ctx, in.UserID, wd); err != nil {
		return store.DayException{}, err
	}
	ex, err := s.q.CreateDayException(ctx, store.CreateDayExceptionParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       in.UserID,
		DepartmentID: target.DepartmentID,
		WorkDate:     wd,
		Code:         in.Code,
		Status:       "approved",
		Source:       "head",
		Note:         in.Note,
		RequestedBy:  store.UUID(headID),
		DecidedBy:    store.UUID(headID),
		DecidedAt:    store.Timestamptz(s.now().UTC()),
		DisplayName:  domain.DisplayName(target.LastName, target.FirstName, target.MiddleName),
	})
	if isUniqueViolation(err) {
		return store.DayException{}, fmt.Errorf("%w: approved exception already exists", domain.ErrConflict)
	}
	return ex, err
}

func (s *Service) DecideAbsence(ctx context.Context, headID, exceptionID uuid.UUID, approve bool) (store.DayException, error) {
	head, err := s.requireSnapshot(ctx, headID)
	if err != nil {
		return store.DayException{}, err
	}
	if !canManageDepartment(head) {
		return store.DayException{}, domain.ErrForbidden
	}
	ex, err := s.q.GetDayExceptionByID(ctx, exceptionID)
	if err != nil {
		return store.DayException{}, mapNoRows(err)
	}
	if store.TextValue(head.DepartmentID) != store.TextValue(ex.DepartmentID) {
		return store.DayException{}, domain.ErrForbidden
	}
	status := "rejected"
	if approve {
		status = "approved"
		if err := s.ensureNoCheckIn(ctx, ex.UserID, ex.WorkDate); err != nil {
			return store.DayException{}, err
		}
	}
	updated, err := s.q.DecideException(ctx, store.DecideExceptionParams{
		ID:        exceptionID,
		Status:    status,
		DecidedBy: store.UUID(headID),
	})
	if isUniqueViolation(err) {
		return store.DayException{}, fmt.Errorf("%w: approved exception already exists", domain.ErrConflict)
	}
	return updated, err
}

func (s *Service) ListPendingAbsences(ctx context.Context, headID uuid.UUID) ([]store.DayException, error) {
	head, err := s.requireSnapshot(ctx, headID)
	if err != nil {
		return nil, err
	}
	if !canManageDepartment(head) {
		return nil, domain.ErrForbidden
	}
	return s.q.ListPendingExceptionsByDept(ctx, store.ListPendingExceptionsByDeptParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: head.DepartmentID,
	})
}

func (s *Service) ensureNoCheckIn(ctx context.Context, userID uuid.UUID, workDate time.Time) error {
	exists, err := s.q.HasDaySession(ctx, store.HasDaySessionParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       userID,
		WorkDate:     workDate,
	})
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrHasCheckIn
	}
	return nil
}
