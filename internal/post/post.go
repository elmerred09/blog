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
	// ReadingMinutes is at least 1.
	ReadingMinutes int
	// Tags are sorted by slug, and never nil.
	Tags []Tag
}

type Post struct {
	ListItem
	BodyHTML  string
	UpdatedAt time.Time
}

// Neighbor is a post linked from the bottom of another post.
type Neighbor struct {
	Slug  string
	Title string
}

// Adjacent is the published posts on either side of one post, in list order.
// A side is nil when the post is the newest or the oldest.
type Adjacent struct {
	Newer *Neighbor
	Older *Neighbor
}

type Page struct {
	// Items is never nil, newest first.
	Items []ListItem
	// Prev is the cursor for the newer page, or nil on the first page.
	Prev *Cursor
	// Next is the cursor for the older page, or nil on the last page.
	Next *Cursor
}
