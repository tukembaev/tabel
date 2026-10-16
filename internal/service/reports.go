package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kstu/tabel/internal/domain"
	"github.com/kstu/tabel/internal/store"
)

type DayReport struct {
	Date          string `json:"date"`
	DayOfWeek     string `json:"day_of_week"`
	Kind          string `json:"kind"` // weekend | absence | attendance | missing | future
	Code          string `json:"code,omitempty"`
	WorkingTime   string `json:"working_time,omitempty"`
	Status        string `json:"status,omitempty"`
	Present       *bool  `json:"present,omitempty"`
	Sessions      int    `json:"sessions,omitempty"`
	GrossSeconds  int64  `json:"gross_seconds,omitempty"`
	LunchSeconds  int64  `json:"lunch_seconds,omitempty"`
	WorkedSeconds int64  `json:"worked_seconds,omitempty"`
	CloseReason   string `json:"close_reason,omitempty"`
	Source        string `json:"source,omitempty"`
	NeedsReview   bool   `json:"needs_review,omitempty"`
	OpenedAt      string `json:"opened_at,omitempty"`
	ClosedAt      string `json:"closed_at,omitempty"`
}

type UserReport struct {
	UserID         uuid.UUID   `json:"user_id"`
	DisplayName    string      `json:"display_name"`
	DepartmentID   string      `json:"department_id,omitempty"`
	DepartmentName string      `json:"department_name,omitempty"`
	Days           []DayReport `json:"days"`
	TotalSeconds   int64       `json:"total_seconds"`
	WorkingDays    int         `json:"working_days"`
	MissingDays    int         `json:"missing_days"`
}

type DepartmentReport struct {
	DepartmentID   string       `json:"department_id"`
	DepartmentName string       `json:"department_name,omitempty"`
	From           string       `json:"from"`
	To             string       `json:"to"`
	Users          []UserReport `json:"users"`
	MissingMarks   int          `json:"missing_marks"`
}

type MissingMark struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Date        string    `json:"date"`
	DayOfWeek   string    `json:"day_of_week"`
}

type DepartmentExport struct {
	DepartmentID   string                    `json:"department_id"`
	DepartmentName string                    `json:"department_name"`
	From           string                    `json:"from"`
	To             string                    `json:"to"`
	Employees      []store.UserSnapshot      `json:"employees"`
	Sessions       []store.AttendanceSession `json:"sessions"`
	Exceptions     []store.DayException      `json:"exceptions"`
	Edits          []store.TimesheetEdit     `json:"edits"`
	Timesheet      DepartmentReport          `json:"timesheet"`
	Missing        []MissingMark             `json:"missing"`
}

var russianWeekdays = map[time.Weekday]string{
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
	time.Sunday:    "Воскресенье",
}

func (s *Service) MyReport(ctx context.Context, userID uuid.UUID, from, to time.Time) (UserReport, error) {
	snap, err := s.requireSnapshot(ctx, userID)
	if err != nil {
		return UserReport{}, err
	}
	return s.buildUserReport(ctx, snap, from, to)
}

func (s *Service) UserReport(ctx context.Context, requester, target uuid.UUID, from, to time.Time) (UserReport, error) {
	req, err := s.requireSnapshot(ctx, requester)
	if err != nil {
		return UserReport{}, err
	}
	if requester != target && !canManageDepartment(req) {
		return UserReport{}, domain.ErrForbidden
	}
	snap, err := s.requireSnapshot(ctx, target)
	if err != nil {
		return UserReport{}, err
	}
	if requester != target && !sameDepartment(req, snap) {
		return UserReport{}, domain.ErrForbidden
	}
	return s.buildUserReport(ctx, snap, from, to)
}

func (s *Service) DepartmentReport(ctx context.Context, editorID uuid.UUID, from, to time.Time) (DepartmentReport, error) {
	editor, err := s.requireDeptManager(ctx, editorID)
	if err != nil {
		return DepartmentReport{}, err
	}
	return s.departmentReportFor(ctx, editor, from, to)
}

func (s *Service) DepartmentReportByID(ctx context.Context, editorID uuid.UUID, departmentID string, from, to time.Time) (DepartmentReport, error) {
	if _, err := s.requireDeptManagerOf(ctx, editorID, departmentID); err != nil {
		return DepartmentReport{}, err
	}
	return s.DepartmentReport(ctx, editorID, from, to)
}

func (s *Service) DepartmentMissing(ctx context.Context, editorID uuid.UUID, departmentID string, from, to time.Time) ([]MissingMark, error) {
	rep, err := s.DepartmentReportByID(ctx, editorID, departmentID, from, to)
	if err != nil {
		return nil, err
	}
	return missingFromReport(rep), nil
}

func (s *Service) DepartmentExport(ctx context.Context, editorID uuid.UUID, departmentID string, from, to time.Time) (DepartmentExport, error) {
	editor, err := s.requireDeptManagerOf(ctx, editorID, departmentID)
	if err != nil {
		return DepartmentExport{}, err
	}
	dept := store.TextValue(editor.DepartmentID)
	employees, err := s.q.ListEmployeesByDepartment(ctx, store.ListEmployeesByDepartmentParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: editor.DepartmentID,
	})
	if err != nil {
		return DepartmentExport{}, err
	}
	sessions, exceptions, err := s.loadDeptRange(ctx, editor.DepartmentID, from, to)
	if err != nil {
		return DepartmentExport{}, err
	}
	edits, err := s.q.ListTimesheetEditsByDept(ctx, store.ListTimesheetEditsByDeptParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: editor.DepartmentID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return DepartmentExport{}, err
	}
	timesheet, err := s.departmentReportFor(ctx, editor, from, to)
	if err != nil {
		return DepartmentExport{}, err
	}
	return DepartmentExport{
		DepartmentID:   dept,
		DepartmentName: editor.DepartmentName,
		From:           s.workDate(from).Format("2006-01-02"),
		To:             s.workDate(to).Format("2006-01-02"),
		Employees:      employees,
		Sessions:       sessions,
		Exceptions:     exceptions,
		Edits:          edits,
		Timesheet:      timesheet,
		Missing:        missingFromReport(timesheet),
	}, nil
}

func missingFromReport(rep DepartmentReport) []MissingMark {
	out := []MissingMark{}
	for _, u := range rep.Users {
		for _, d := range u.Days {
			if d.NeedsReview {
				out = append(out, MissingMark{
					UserID:      u.UserID,
					DisplayName: u.DisplayName,
					Date:        d.Date,
					DayOfWeek:   d.DayOfWeek,
				})
			}
		}
	}
	return out
}

func (s *Service) loadDeptRange(ctx context.Context, departmentID pgtype.Text, from, to time.Time) ([]store.AttendanceSession, []store.DayException, error) {
	sessions, err := s.q.ListDepartmentSessionsInRange(ctx, store.ListDepartmentSessionsInRangeParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: departmentID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return nil, nil, err
	}
	arch, err := s.q.ListDepartmentSessionsArchiveInRange(ctx, store.ListDepartmentSessionsArchiveInRangeParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: departmentID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return nil, nil, err
	}
	for _, a := range arch {
		sessions = append(sessions, store.AttendanceSession(a))
	}
	exceptions, err := s.q.ListDepartmentExceptionsInRange(ctx, store.ListDepartmentExceptionsInRangeParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: departmentID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return nil, nil, err
	}
	earch, err := s.q.ListDepartmentExceptionsArchiveInRange(ctx, store.ListDepartmentExceptionsArchiveInRangeParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: departmentID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return nil, nil, err
	}
	for _, a := range earch {
		exceptions = append(exceptions, store.DayException(a))
	}
	return sessions, exceptions, nil
}

func (s *Service) departmentReportFor(ctx context.Context, editor store.UserSnapshot, from, to time.Time) (DepartmentReport, error) {
	dept := store.TextValue(editor.DepartmentID)
	employees, err := s.q.ListEmployeesByDepartment(ctx, store.ListEmployeesByDepartmentParams{
		UniversityID: s.cfg.UniversityID,
		DepartmentID: editor.DepartmentID,
	})
	if err != nil {
		return DepartmentReport{}, err
	}
	sessions, exceptions, err := s.loadDeptRange(ctx, editor.DepartmentID, from, to)
	if err != nil {
		return DepartmentReport{}, err
	}

	byUser := map[uuid.UUID]*userAgg{}
	for _, emp := range employees {
		byUser[emp.UserID] = &userAgg{
			id:   emp.UserID,
			name: domain.DisplayName(emp.LastName, emp.FirstName, emp.MiddleName),
			dept: dept,
		}
	}
	for _, sess := range sessions {
		u := byUser[sess.UserID]
		if u == nil {
			u = &userAgg{id: sess.UserID, name: sess.DisplayName, dept: dept}
			byUser[sess.UserID] = u
		}
		u.sessions = append(u.sessions, sess)
	}
	for _, ex := range exceptions {
		u := byUser[ex.UserID]
		if u == nil {
			u = &userAgg{id: ex.UserID, name: ex.DisplayName, dept: dept}
			byUser[ex.UserID] = u
		}
		u.exceptions = append(u.exceptions, ex)
	}

	out := DepartmentReport{
		DepartmentID:   dept,
		DepartmentName: editor.DepartmentName,
		From:           s.workDate(from).Format("2006-01-02"),
		To:             s.workDate(to).Format("2006-01-02"),
		Users:          make([]UserReport, 0, len(byUser)),
	}
	for _, u := range byUser {
		rep := s.composeDays(u.id, u.name, dept, editor.DepartmentName, u.sessions, u.exceptions, from, to)
		out.MissingMarks += rep.MissingDays
		out.Users = append(out.Users, rep)
	}
	return out, nil
}

type userAgg struct {
	id         uuid.UUID
	name       string
	dept       string
	sessions   []store.AttendanceSession
	exceptions []store.DayException
}

func (s *Service) buildUserReport(ctx context.Context, snap store.UserSnapshot, from, to time.Time) (UserReport, error) {
	sessions, err := s.q.ListUserSessionsInRange(ctx, store.ListUserSessionsInRangeParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       snap.UserID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return UserReport{}, err
	}
	arch, err := s.q.ListUserSessionsArchiveInRange(ctx, store.ListUserSessionsArchiveInRangeParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       snap.UserID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return UserReport{}, err
	}
	for _, a := range arch {
		sessions = append(sessions, store.AttendanceSession(a))
	}
	exceptions, err := s.q.ListUserExceptionsInRange(ctx, store.ListUserExceptionsInRangeParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       snap.UserID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return UserReport{}, err
	}
	earch, err := s.q.ListUserExceptionsArchiveInRange(ctx, store.ListUserExceptionsArchiveInRangeParams{
		UniversityID: s.cfg.UniversityID,
		UserID:       snap.UserID,
		WorkDate:     s.workDate(from),
		WorkDate_2:   s.workDate(to),
	})
	if err != nil {
		return UserReport{}, err
	}
	for _, a := range earch {
		exceptions = append(exceptions, store.DayException(a))
	}
	return s.composeDays(
		snap.UserID,
		domain.DisplayName(snap.LastName, snap.FirstName, snap.MiddleName),
		store.TextValue(snap.DepartmentID),
		snap.DepartmentName,
		sessions,
		exceptions,
		from,
		to,
	), nil
}

func (s *Service) composeDays(
	userID uuid.UUID,
	name, deptID, deptName string,
	sessions []store.AttendanceSession,
	exceptions []store.DayException,
	from, to time.Time,
) UserReport {
	fromD := s.workDate(from)
	toD := s.workDate(to)
	now := s.workDate(s.now())

	exByDay := map[string]store.DayException{}
	for _, ex := range exceptions {
		if ex.Status != "approved" {
			continue
		}
		exByDay[ex.WorkDate.Format("2006-01-02")] = ex
	}
	sessByDay := map[string][]store.AttendanceSession{}
	for _, sess := range sessions {
		key := sess.WorkDate.Format("2006-01-02")
		sessByDay[key] = append(sessByDay[key], sess)
	}

	var days []DayReport
	var total int64
	workingDays := 0
	missingDays := 0

	for d := fromD; !d.After(toD); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dr := DayReport{
			Date:      key,
			DayOfWeek: russianWeekdays[d.Weekday()],
		}
		if code, ok := domain.IsWeekend(d); ok {
			dr.Kind = "weekend"
			dr.Code = code
			dr.WorkingTime = "Выходной"
			days = append(days, dr)
			continue
		}
		workingDays++
		if d.After(now) {
			dr.Kind = "future"
			dr.WorkingTime = "День не наступил"
			days = append(days, dr)
			continue
		}
		if ex, ok := exByDay[key]; ok {
			dr.Kind = "absence"
			dr.Code = ex.Code
			dr.WorkingTime = ex.Code
			days = append(days, dr)
			continue
		}
		list := sessByDay[key]
		if len(list) == 0 {
			dr.Kind = "missing"
			dr.WorkingTime = "Нет отметки"
			dr.NeedsReview = true
			missingDays++
			days = append(days, dr)
			continue
		}
		dr.Kind = "attendance"
		dr.Sessions = len(list)
		var gross, lunch int64
		present := false
		for _, sess := range list {
			dr.Status = sess.Status
			dr.Source = sess.Source
			dr.CloseReason = store.TextValue(sess.CloseReason)
			dr.OpenedAt = sess.OpenedAt.Format(time.RFC3339)
			if sess.ClosedAt.Valid {
				dr.ClosedAt = sess.ClosedAt.Time.Format(time.RFC3339)
			}
			if sess.DurationSeconds.Valid {
				gross += int64(sess.DurationSeconds.Int32)
			}
			lunch += int64(sess.LunchSeconds)
			if sess.Status == "closed" || sess.Status == "open" {
				present = true
			}
		}
		worked := gross - lunch
		if worked < 0 {
			worked = 0
		}
		dr.GrossSeconds = gross
		dr.LunchSeconds = lunch
		dr.WorkedSeconds = worked
		if worked > 0 {
			total += worked
			dr.WorkingTime = formatDuration(worked)
		} else if present {
			t := true
			dr.Present = &t
			dr.WorkingTime = "Присутствовал"
		} else {
			f := false
			dr.Present = &f
			dr.WorkingTime = "Нет отметки"
		}
		days = append(days, dr)
	}

	return UserReport{
		UserID:         userID,
		DisplayName:    name,
		DepartmentID:   deptID,
		DepartmentName: deptName,
		Days:           days,
		TotalSeconds:   total,
		WorkingDays:    workingDays,
		MissingDays:    missingDays,
	}
}

func formatDuration(sec int64) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	return time.Date(0, 1, 1, int(h), int(m), int(s), 0, time.UTC).Format("15:04:05")
}
