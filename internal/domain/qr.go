package domain

import (
	"fmt"
	"net/url"
	"strings"
)

const QRHost = "qr.kstu.kg"

type QRLocation struct {
	UniversityID string
	BuildingID   string
	RoomNumber   string
}

// ParseQRURL extracts university, building and room from a campus QR URL.
// Expected form: https://qr.kstu.kg/kstu/1/102
func ParseQRURL(raw string) (QRLocation, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return QRLocation{}, fmt.Errorf("%w: qr_url is required", ErrInvalidQR)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return QRLocation{}, fmt.Errorf("%w: invalid url", ErrInvalidQR)
	}
	if !strings.EqualFold(u.Hostname(), QRHost) {
		return QRLocation{}, fmt.Errorf("%w: unexpected host", ErrInvalidQR)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return QRLocation{}, fmt.Errorf("%w: unexpected scheme", ErrInvalidQR)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return QRLocation{}, fmt.Errorf("%w: expected /{university}/{building}/{room}", ErrInvalidQR)
	}
	return QRLocation{
		UniversityID: parts[0],
		BuildingID:   parts[1],
		RoomNumber:   parts[2],
	}, nil
}
