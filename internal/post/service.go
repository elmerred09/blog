package post

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/elmerred09/blog/internal/db"
)

const (
	// DefaultLimit is the page size callers should use when none is given.
	DefaultLimit = 10
	// MaxLimit is the largest page size List accepts.
	MaxLimit = 50
)

// ErrInvalidLimit means ListParams.Limit is outside 1 to MaxLimit.
var ErrInvalidLimit = fmt.Errorf("limit must be between 1 and %d", MaxLimit)

type Service struct {
	q *db.Queries
}

func NewService(conn db.DBTX) *Service {
	return &Service{q: db.New(conn)}
}

type ListParams struct {
	// Cursor is the previous page's Prev or Next, or nil for the first page.
	Cursor *Cursor
	// Limit is the page size, from 1 to MaxLimit.
	Limit int
}

// List returns one page of published posts, newest first, with their tags.
func (s *Service) List(ctx context.Context, p ListParams) (Page, error) {
	if p.Limit < 1 || p.Limit > MaxLimit {
		return Page{}, fmt.Errorf("%w: got %d", ErrInvalidLimit, p.Limit)
	}
	if p.Cursor != nil && p.Cursor.Direction == Before {
		return s.listBefore(ctx, p)
	}
	return s.listAfter(ctx, p)
}

// listAfter is a page of posts older than the cursor (or the newest posts,
// when there is no cursor), read in the list order.
func (s *Service) listAfter(ctx context.Context, p ListParams) (Page, error) {
	// One extra row says whether another page exists, without a count query.
	params := db.ListPostsParams{PageSize: int32(p.Limit + 1)}
	if p.Cursor != nil {
		params.AfterPublishedAt = &p.Cursor.PublishedAt
		params.AfterID = &p.Cursor.ID
	}
	rows, err := s.q.ListPosts(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}

	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}
	// A cursor was given, so something newer than this page exists: the row
	// the cursor itself points at.
	return s.pageFrom(ctx, listRows(rows), p.Cursor != nil, hasMore)
}

// listBefore is a page of posts newer than the cursor. The query returns
// them oldest first; they're reversed so the page reads newest first, like
// every other page. A short result has reached the top, and the first page
// replaces it, so arriving there shows exactly what no cursor shows.
func (s *Service) listBefore(ctx context.Context, p ListParams) (Page, error) {
	rows, err := s.q.ListPostsBefore(ctx, db.ListPostsBeforeParams{
		BeforePublishedAt: p.Cursor.PublishedAt,
		BeforeID:          p.Cursor.ID,
		PageSize:          int32(p.Limit + 1),
	})
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	if len(rows) <= p.Limit {
		return s.listAfter(ctx, ListParams{Limit: p.Limit})
	}
	rows = rows[:p.Limit]
	slices.Reverse(rows)
	return s.pageFrom(ctx, listBeforeRows(rows), true, true)
}

// pageFrom attaches tags and the cursors. withPrev and withNext say whether
// each direction has another page; the cursors point at this page's ends.
func (s *Service) pageFrom(ctx context.Context, rows []Row, withPrev, withNext bool) (Page, error) {
	items, err := ListItems(ctx, s.q, rows)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	page := Page{Items: items}
	if withPrev && len(rows) > 0 {
		first := rows[0]
		page.Prev = &Cursor{Direction: Before, PublishedAt: *first.PublishedAt, ID: first.ID}
	}
	if withNext && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.Next = &Cursor{Direction: After, PublishedAt: *last.PublishedAt, ID: last.ID}
	}
	return page, nil
}

func listRows(rows []db.ListPostsRow) []Row {
	out := make([]Row, len(rows))
	for i, r := range rows {
		out[i] = Row(r)
	}
	return out
}

func listBeforeRows(rows []db.ListPostsBeforeRow) []Row {
	out := make([]Row, len(rows))
	for i, r := range rows {
		out[i] = Row(r)
	}
	return out
}

// Row is the columns a list-style query (ListPosts, SearchPosts) selects for
// each post.
type Row struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Summary     string
	PublishedAt *time.Time
}

// ListItems turns rows into ListItems in the same order, with each post's tags
// from one query. Every row must be published.
func ListItems(ctx context.Context, q *db.Queries, rows []Row) ([]ListItem, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	tagsByPost, err := tagsFor(ctx, q, ids)
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, len(rows))
	for i, r := range rows {
		if r.PublishedAt == nil {
			return nil, fmt.Errorf("post %s has no published_at", r.ID)
		}
		tags := tagsByPost[r.ID]
		if tags == nil {
			tags = []Tag{}
		}
		items[i] = ListItem{
			Slug:        r.Slug,
			Title:       r.Title,
			Summary:     r.Summary,
			PublishedAt: *r.PublishedAt,
			Tags:        tags,
		}
	}
	return items, nil
}

// tagsFor returns each post's tags, sorted by slug. Posts without tags are
// absent from the map.
func tagsFor(ctx context.Context, q *db.Queries, postIDs []uuid.UUID) (map[uuid.UUID][]Tag, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}
	rows, err := q.ListTagsForPosts(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("list tags for posts: %w", err)
	}
	// Rows arrive sorted by slug, so appending keeps each post's tags sorted.
	tagsByPost := make(map[uuid.UUID][]Tag, len(postIDs))
	for _, r := range rows {
		tagsByPost[r.PostID] = append(tagsByPost[r.PostID], Tag{Slug: r.Slug, Name: r.Name})
	}
	return tagsByPost, nil
}

// Get returns the published post with this slug, or ErrNotFound.
func (s *Service) Get(ctx context.Context, slug string) (Post, error) {
	row, err := s.q.GetPostBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, fmt.Errorf("get post %q: %w", slug, err)
	}
	if row.PublishedAt == nil {
		return Post{}, fmt.Errorf("get post %q: no published_at", slug)
	}

	tagRows, err := s.q.ListTagsForPost(ctx, row.ID)
	if err != nil {
		return Post{}, fmt.Errorf("list tags for post %q: %w", slug, err)
	}
	tags := make([]Tag, len(tagRows))
	for i, r := range tagRows {
		tags[i] = Tag{Slug: r.Slug, Name: r.Name}
	}

	return Post{
		Slug:        row.Slug,
		Title:       row.Title,
		Summary:     row.Summary,
		PublishedAt: *row.PublishedAt,
		Tags:        tags,
		BodyHTML:    row.BodyHtml,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}
