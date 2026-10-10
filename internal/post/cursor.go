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

// Cursor marks the last item of a page; the next page starts after it.
// The list order is (published_at, id) descending, so both are needed to
// break ties between posts published at the same instant.
type Cursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

const cursorSep = "|"

// String encodes the cursor for clients, who must treat it as opaque.
// RFC3339Nano keeps Postgres's microseconds, so the next page neither skips
// nor repeats a post.
func (c Cursor) String() string {
	raw := c.PublishedAt.UTC().Format(time.RFC3339Nano) + cursorSep + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// ParseCursor decodes a cursor made by Cursor.String. Every failure wraps
// ErrInvalidCursor.
func ParseCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %w", ErrInvalidCursor, err)
	}

	at, id, ok := strings.Cut(string(raw), cursorSep)
	if !ok {
		return Cursor{}, fmt.Errorf("%w: missing separator", ErrInvalidCursor)
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

	return Cursor{PublishedAt: publishedAt, ID: parsedID}, nil
}
