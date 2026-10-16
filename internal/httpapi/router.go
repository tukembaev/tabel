package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/kstu/tabel/internal/auth"
	"github.com/kstu/tabel/internal/domain"
	"github.com/kstu/tabel/internal/httpapi/middleware"
	"github.com/kstu/tabel/internal/service"
)

type API struct {
	svc       *service.Service
	jwt       *auth.Verifier
	authCfg   middleware.AuthConfig
	basicUser string
	basicPass string
}

func New(svc *service.Service, jwt *auth.Verifier, authCfg middleware.AuthConfig, basicUser, basicPass string) *API {
	return &API{svc: svc, jwt: jwt, authCfg: authCfg, basicUser: basicUser, basicPass: basicPass}
}

func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Sets HttpOnly cookie access_token for browser / Swagger (credentials: include).
	r.Post("/auth/login", a.login)
	r.Post("/auth/logout", a.logout)

	mountSwagger(r)

	r.Route("/v1", func(r chi.Router) {
		r.Use(middleware.JWT(a.jwt, a.authCfg))
		r.Post("/me/sync", a.syncMe)

		r.Post("/attendance/scan", a.scan)

		r.Put("/timesheet/days", a.editTimesheetDay)
		r.Get("/timesheet/edits", a.listTimesheetEdits)

		r.Get("/departments/{departmentID}/employees", a.deptEmployees)
		r.Get("/departments/{departmentID}/export", a.deptExport)
		r.Get("/departments/{departmentID}/missing", a.deptMissing)

		r.Post("/absences/requests", a.requestAbsence)
		r.Get("/absences/pending", a.pendingAbsences)
		r.Post("/absences/head", a.headSetAbsence)
		r.Post("/absences/{id}/decide", a.decideAbsence)

		r.Get("/reports/me", a.reportMe)
		r.Get("/reports/users/{userID}", a.reportUser)
		r.Get("/reports/department", a.reportDepartment)
		r.Get("/reports/departments/{departmentID}", a.reportDepartmentByID)
	})
	return r
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	if a.basicUser == "" || body.Username != a.basicUser || body.Password != a.basicPass {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if a.authCfg.BasicUserID == uuid.Nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "dev user not configured"})
		return
	}
	tok, err := a.jwt.SignAccessToken(a.authCfg.BasicUserID, 24*time.Hour)
	if err != nil {
		writeErr(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.AccessCookie,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 3600,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      tok,
		"user_id":    a.authCfg.BasicUserID.String(),
		"expires_in": 24 * 3600,
	})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.AccessCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) syncMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeErr(w, domain.ErrUnauthorized)
		return
	}
	var in service.SyncInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	snap, err := a.svc.SyncMe(r.Context(), uid, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (a *API) scan(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	var in service.ScanInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	res, err := a.svc.Scan(r.Context(), uid, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	code := http.StatusOK
	if res.Action == service.ScanActionCheckIn {
		code = http.StatusCreated
	}
	writeJSON(w, code, res)
}

func (a *API) editTimesheetDay(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		UserID   string `json:"user_id"`
		WorkDate string `json:"work_date"`
		Kind     string `json:"kind"`
		OpenedAt string `json:"opened_at"`
		ClosedAt string `json:"closed_at"`
		Code     string `json:"code"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	target, err := uuid.Parse(body.UserID)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	wd, err := parseAPIDate(body.WorkDate)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	opened, err := parseOptionalRFC3339(body.OpenedAt)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	closed, err := parseOptionalRFC3339(body.ClosedAt)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	res, err := a.svc.EditDepartmentDay(r.Context(), uid, service.EditDayInput{
		UserID:   target,
		WorkDate: wd,
		Kind:     body.Kind,
		OpenedAt: opened,
		ClosedAt: closed,
		Code:     body.Code,
		Note:     body.Note,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (a *API) listTimesheetEdits(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	list, err := a.svc.ListDepartmentEdits(r.Context(), uid, r.URL.Query().Get("department_id"), from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) deptEmployees(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	list, err := a.svc.ListDepartmentEmployees(r.Context(), uid, chi.URLParam(r, "departmentID"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) deptExport(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := a.svc.DepartmentExport(r.Context(), uid, chi.URLParam(r, "departmentID"), from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (a *API) deptMissing(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	list, err := a.svc.DepartmentMissing(r.Context(), uid, chi.URLParam(r, "departmentID"), from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) requestAbsence(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		WorkDate string `json:"work_date"`
		Code     string `json:"code"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	wd, err := parseAPIDate(body.WorkDate)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	ex, err := a.svc.RequestAbsence(r.Context(), uid, service.AbsenceRequestInput{
		WorkDate: wd,
		Code:     body.Code,
		Note:     body.Note,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ex)
}

func (a *API) pendingAbsences(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	list, err := a.svc.ListPendingAbsences(r.Context(), uid)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (a *API) headSetAbsence(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		UserID   string `json:"user_id"`
		WorkDate string `json:"work_date"`
		Code     string `json:"code"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	target, err := uuid.Parse(body.UserID)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	wd, err := parseAPIDate(body.WorkDate)
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	ex, err := a.svc.HeadSetAbsence(r.Context(), uid, service.HeadSetAbsenceInput{
		UserID:   target,
		WorkDate: wd,
		Code:     body.Code,
		Note:     body.Note,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ex)
}

func (a *API) decideAbsence(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	var body struct {
		Approve bool `json:"approve"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	ex, err := a.svc.DecideAbsence(r.Context(), uid, id, body.Approve)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

func (a *API) reportMe(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := a.svc.MyReport(r.Context(), uid, from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (a *API) reportUser(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeErr(w, fmtValidation(err))
		return
	}
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := a.svc.UserReport(r.Context(), uid, target, from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (a *API) reportDepartment(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := a.svc.DepartmentReport(r.Context(), uid, from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (a *API) reportDepartmentByID(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())
	from, to, err := parseRange(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	rep, err := a.svc.DepartmentReportByID(r.Context(), uid, chi.URLParam(r, "departmentID"), from, to)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func parseRange(r *http.Request) (time.Time, time.Time, error) {
	fromS := r.URL.Query().Get("from")
	toS := r.URL.Query().Get("to")
	if fromS == "" || toS == "" {
		now := time.Now()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to := from.AddDate(0, 1, -1)
		return from, to, nil
	}
	from, err := time.Parse("2006-01-02", fromS)
	if err != nil {
		return time.Time{}, time.Time{}, fmtValidation(err)
	}
	to, err := time.Parse("2006-01-02", toS)
	if err != nil {
		return time.Time{}, time.Time{}, fmtValidation(err)
	}
	return from, to, nil
}

func fmtValidation(err error) error {
	return errors.Join(domain.ErrValidation, err)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	msg := err.Error()
	switch {
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrSnapshotRequired):
		code = http.StatusUnauthorized
		if errors.Is(err, domain.ErrSnapshotRequired) {
			code = http.StatusPreconditionRequired
		}
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrHeadOnlyCode):
		code = http.StatusForbidden
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrNoOpenSession):
		code = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrHasCheckIn):
		code = http.StatusConflict
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrAfterCutoff), errors.Is(err, domain.ErrInvalidCode), errors.Is(err, domain.ErrSessionClosed), errors.Is(err, domain.ErrInvalidQR), errors.Is(err, domain.ErrDayAlreadyClosed):
		code = http.StatusBadRequest
	}
	writeJSON(w, code, map[string]string{"error": msg})
}
