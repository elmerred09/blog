package post_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/post"
)

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()

	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	tests := []struct {
		name string
		at   time.Time
	}{
		{"microseconds survive", time.Date(2026, 10, 1, 12, 30, 45, 123456000, time.UTC)},
		{"whole seconds", time.Date(2026, 10, 1, 12, 30, 45, 0, time.UTC)},
		{"non-UTC zone keeps the instant", time.Date(2026, 10, 1, 8, 30, 45, 1000, newYork)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := post.Cursor{PublishedAt: tc.at, ID: uuid.Must(uuid.NewV7())}

			s := want.String()
			if strings.ContainsAny(s, "+/=") {
				t.Errorf("cursor %q is not unpadded base64url", s)
			}

			got, err := post.ParseCursor(s)
			if err != nil {
				t.Fatalf("ParseCursor(%q): %v", s, err)
			}
			if !got.PublishedAt.Equal(want.PublishedAt) || got.ID != want.ID {
				t.Errorf("round trip = %+v, want %+v", got, want)
			}
		})
	}
}

func TestParseCursorRejects(t *testing.T) {
	t.Parallel()

	encode := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	const (
		validTime = "2026-10-01T12:30:45.123456Z"
		validID   = "01a10933-7891-7d22-8e01-7c250fa9aebf"
	)

	tests := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"not base64", "not a cursor!"},
		{"padded base64", base64.URLEncoding.EncodeToString([]byte(validTime + "|" + validID))}, // 64 bytes, so "==" padding
		{"missing separator", encode(validTime + validID)},
		{"bad time", encode("yesterday|" + validID)},
		{"time without zone", encode("2026-10-01T12:30:45|" + validID)},
		{"bad uuid", encode(validTime + "|not-a-uuid")},
		{"non-canonical uuid", encode(validTime + "|" + strings.ReplaceAll(validID, "-", ""))},
		{"extra field", encode(validTime + "|" + validID + "|more")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := post.ParseCursor(tc.cursor)
			if !errors.Is(err, post.ErrInvalidCursor) {
				t.Errorf("ParseCursor(%q) = %+v, %v; want ErrInvalidCursor", tc.cursor, got, err)
			}
		})
	}
}
