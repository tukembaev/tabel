package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kstu/tabel/internal/auth"
)

// Usage: go run ./scripts/gentoken.go [user-uuid]
func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	sub := "15b7fbe4-95d0-4319-a6c5-fa1ea13391ed"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sub,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	s, err := tok.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
}
