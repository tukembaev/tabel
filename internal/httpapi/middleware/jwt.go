package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/kstu/tabel/internal/auth"
)

// AuthConfig controls cookie + Basic Auth fallbacks used by Swagger / local testing.
type AuthConfig struct {
	// BasicUser / BasicPassword enable Authorization: Basic for Swagger Authorize (default user/user).
	BasicUser     string
	BasicPassword string
	// BasicUserID is the UUID used when Basic Auth succeeds.
	BasicUserID uuid.UUID
}

func JWT(v *auth.Verifier, cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := resolveUser(r, v, cfg)
			if !ok {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			ctx := auth.WithUserID(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func resolveUser(r *http.Request, v *auth.Verifier, cfg AuthConfig) (uuid.UUID, bool) {
	// 1) Cookie (browser / Swagger after /auth/login)
	if c, err := r.Cookie(auth.AccessCookie); err == nil && strings.TrimSpace(c.Value) != "" {
		if id, err := v.ParseAccessToken(strings.TrimSpace(c.Value)); err == nil {
			return id, true
		}
	}

	h := r.Header.Get("Authorization")
	if h == "" {
		return uuid.Nil, false
	}
	lower := strings.ToLower(h)

	// 2) Bearer JWT
	if strings.HasPrefix(lower, "bearer ") {
		token := strings.TrimSpace(h[7:])
		id, err := v.ParseAccessToken(token)
		if err != nil {
			return uuid.Nil, false
		}
		return id, true
	}

	// 3) Basic user:user → fixed test UUID (Swagger Authorize)
	if strings.HasPrefix(lower, "basic ") && cfg.BasicUser != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(h[6:]))
		if err != nil {
			return uuid.Nil, false
		}
		user, pass, ok := strings.Cut(string(raw), ":")
		if !ok || user != cfg.BasicUser || pass != cfg.BasicPassword {
			return uuid.Nil, false
		}
		if cfg.BasicUserID == uuid.Nil {
			return uuid.Nil, false
		}
		return cfg.BasicUserID, true
	}

	return uuid.Nil, false
}
