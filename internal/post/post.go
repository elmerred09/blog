package post

import (
	"errors"
	"time"

	"github.com/elmerred09/blog/internal/tags"
)

var ErrNotFound = errors.New("post not found")

type Tag = tags.Tag

// ListItem is a post as it appears in a list: everything but the body.
type ListItem struct {
	Slug        string
	Title       string
	Summary     string
	PublishedAt time.Time
	// Tags are sorted by slug, and never nil.
	Tags []Tag
}

type Post struct {
	ListItem
	BodyHTML  string
	UpdatedAt time.Time
}

type Page struct {
	// Items is never nil, newest first.
	Items []ListItem
	// Prev is the cursor for the newer page, or nil on the first page.
	Prev *Cursor
	// Next is the cursor for the older page, or nil on the last page.
	Next *Cursor
}
