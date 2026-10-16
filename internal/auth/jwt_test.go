package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kstu/tabel/internal/auth"
)

func TestParseAccessToken(t *testing.T) {
	secret := "test-secret"
	uid := uuid.MustParse("15b7fbe4-95d0-4319-a6c5-fa1ea13391ed")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uid.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	got, err := auth.NewVerifier(secret).ParseAccessToken(signed)
	if err != nil {
		t.Fatal(err)
	}
	if got != uid {
		t.Fatalf("got %s want %s", got, uid)
	}
}
