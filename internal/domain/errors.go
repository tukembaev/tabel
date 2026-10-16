package domain

import "errors"

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrValidation       = errors.New("validation")
	ErrSnapshotRequired = errors.New("user snapshot required; call POST /v1/me/sync")
	ErrInvalidQR        = errors.New("invalid qr code")
	ErrAfterCutoff      = errors.New("check-in after staff cutoff is not allowed")
	ErrNoOpenSession    = errors.New("no open session")
	ErrSessionClosed    = errors.New("session is closed")
	ErrHasCheckIn       = errors.New("cannot set absence code: check-in already exists")
	ErrInvalidCode      = errors.New("invalid absence code")
	ErrHeadOnlyCode     = errors.New("this code can only be set by department head")
	ErrDayAlreadyClosed = errors.New("work day is already closed")
)
