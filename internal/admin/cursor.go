package admin

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ErrInvalidCursor means a cursor string couldn't be parsed.
var ErrInvalidCursor = errors.New("invalid cursor")

// Direction says which way a cursor pages. After is the older page (lower
// ids, since ids are uuidv7 and the list is newest created first); Before
// is the newer page.
type Direction int

const (
	After Direction = iota
	Before
)

// Cursor marks a post id in the list order. Admin URLs aren't shared, so
// there is no legacy form without the direction prefix.
type Cursor struct {
	Direction Direction
	ID        uuid.UUID
}

// String encodes the cursor for clients, who must treat it as opaque.
func (c Cursor) String() string {
	dir := "a"
	if c.Direction == Before {
		dir = "b"
	}
	raw := dir + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// ParseCursor decodes a cursor made by Cursor.String. Every failure wraps
// ErrInvalidCursor.
func ParseCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}

	dir, id, ok := strings.Cut(string(raw), "|")
	if !ok || id == "" {
		return Cursor{}, fmt.Errorf("%w: missing id", ErrInvalidCursor)
	}
	c := Cursor{}
	switch dir {
	case "a":
		c.Direction = After
	case "b":
		c.Direction = Before
	default:
		return Cursor{}, fmt.Errorf("%w: unknown direction %q", ErrInvalidCursor, dir)
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}
	// uuid.Parse also accepts forms String never produces (no hyphens, braces,
	// urn:uuid:), which would give one cursor many spellings.
	if parsed.String() != id {
		return Cursor{}, fmt.Errorf("%w: id %q is not in canonical form", ErrInvalidCursor, id)
	}
	c.ID = parsed
	return c, nil
}
