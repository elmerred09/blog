package post

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCursor means a cursor string couldn't be parsed.
var ErrInvalidCursor = errors.New("invalid cursor")

// Direction says which way a cursor pages. After is the older page (the
// default, and what a cursor without a direction means); Before is the newer
// one. The list order is newest first, so After walks down it and Before up.
type Direction int

const (
	After Direction = iota
	Before
)

// Cursor marks an item on the (published_at, id) order. Both fields are
// needed to break ties between posts published at the same instant.
type Cursor struct {
	Direction   Direction
	PublishedAt time.Time
	ID          uuid.UUID
}

const cursorSep = "|"

// String encodes the cursor for clients, who must treat it as opaque.
// The direction is a prefix, "a" or "b", and RFC3339Nano keeps Postgres's
// microseconds, so the next page neither skips nor repeats a post.
func (c Cursor) String() string {
	dir := "a"
	if c.Direction == Before {
		dir = "b"
	}
	raw := dir + cursorSep + c.PublishedAt.UTC().Format(time.RFC3339Nano) + cursorSep + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// ParseCursor decodes a cursor made by Cursor.String. A cursor from before
// directions existed, "time|id" with no prefix, is an After cursor, so old
// links keep working. Every failure wraps ErrInvalidCursor.
func ParseCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}

	parts := strings.Split(string(raw), cursorSep)
	var (
		c      Cursor
		at, id string
	)
	switch len(parts) {
	case 2:
		c.Direction = After
		at, id = parts[0], parts[1]
	case 3:
		switch parts[0] {
		case "a":
			c.Direction = After
		case "b":
			c.Direction = Before
		default:
			return Cursor{}, fmt.Errorf("%w: unknown direction %q", ErrInvalidCursor, parts[0])
		}
		at, id = parts[1], parts[2]
	default:
		return Cursor{}, fmt.Errorf("%w: want a direction, a time and an id", ErrInvalidCursor)
	}

	publishedAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}
	// uuid.Parse also accepts forms String never produces (no hyphens, braces,
	// urn:uuid:), which would give one cursor many spellings.
	if parsedID.String() != id {
		return Cursor{}, fmt.Errorf("%w: id %q is not in canonical form", ErrInvalidCursor, id)
	}

	c.PublishedAt = publishedAt
	c.ID = parsedID
	return c, nil
}
