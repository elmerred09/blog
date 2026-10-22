package admin_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/admin"
)

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()

	for _, dir := range []admin.Direction{admin.After, admin.Before} {
		want := admin.Cursor{Direction: dir, ID: uuid.Must(uuid.NewV7())}

		s := want.String()
		if strings.ContainsAny(s, "+/=") {
			t.Errorf("cursor %q is not unpadded base64url", s)
		}
		got, err := admin.ParseCursor(s)
		if err != nil {
			t.Fatalf("ParseCursor(%q): %v", s, err)
		}
		if got != want {
			t.Errorf("round trip = %+v, want %+v", got, want)
		}
	}
}

func TestParseCursorRejects(t *testing.T) {
	t.Parallel()

	encode := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	const validID = "01a10933-7891-7d22-8e01-7c250fa9aebf"

	tests := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"not base64", "not a cursor!"},
		{"bare uuid", validID},
		{"missing id", encode("a|")},
		{"missing separator", encode("a" + validID)},
		{"unknown direction", encode("c|" + validID)},
		{"bad uuid", encode("a|not-a-uuid")},
		{"non-canonical uuid", encode("a|" + strings.ReplaceAll(validID, "-", ""))},
		{"extra field", encode("a|" + validID + "|more")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := admin.ParseCursor(tc.cursor)
			if !errors.Is(err, admin.ErrInvalidCursor) {
				t.Errorf("ParseCursor(%q) = %+v, %v; want ErrInvalidCursor", tc.cursor, got, err)
			}
		})
	}
}
