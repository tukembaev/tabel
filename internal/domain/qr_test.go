package domain_test

import (
	"errors"
	"testing"

	"github.com/kstu/tabel/internal/domain"
)

func TestParseQRURL(t *testing.T) {
	got, err := domain.ParseQRURL("https://qr.kstu.kg/kstu/1/102")
	if err != nil {
		t.Fatal(err)
	}
	if got.UniversityID != "kstu" || got.BuildingID != "1" || got.RoomNumber != "102" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseQRURL_TrimAndQuery(t *testing.T) {
	got, err := domain.ParseQRURL("  https://qr.kstu.kg/kstu/2/305?x=1  ")
	if err != nil {
		t.Fatal(err)
	}
	if got.BuildingID != "2" || got.RoomNumber != "305" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseQRURL_Rejects(t *testing.T) {
	cases := []string{
		"",
		"not-a-url",
		"https://example.com/kstu/1/102",
		"https://qr.kstu.kg/kstu/1",
		"https://qr.kstu.kg/kstu/1/102/extra",
		"ftp://qr.kstu.kg/kstu/1/102",
	}
	for _, raw := range cases {
		if _, err := domain.ParseQRURL(raw); !errors.Is(err, domain.ErrInvalidQR) {
			t.Errorf("%q: want ErrInvalidQR, got %v", raw, err)
		}
	}
}
