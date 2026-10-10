package post

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("post not found")

type Tag struct {
	Slug string
	Name string
}

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
	// Items is never nil.
	Items []ListItem
	// Next is the cursor for the following page, or nil on the last page.
	Next *Cursor
}
