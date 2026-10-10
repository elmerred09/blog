package post

import (
	"context"
	"errors"
	"fmt"

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
	// After is the previous page's Next, or nil for the first page.
	After *Cursor
	// Limit is the page size, from 1 to MaxLimit.
	Limit int
}

// List returns one page of published posts, newest first, with their tags.
func (s *Service) List(ctx context.Context, p ListParams) (Page, error) {
	if p.Limit < 1 || p.Limit > MaxLimit {
		return Page{}, fmt.Errorf("%w: got %d", ErrInvalidLimit, p.Limit)
	}

	// One extra row says whether another page exists, without a count query.
	params := db.ListPostsParams{PageSize: int32(p.Limit + 1)}
	if p.After != nil {
		params.AfterPublishedAt = &p.After.PublishedAt
		params.AfterID = &p.After.ID
	}
	rows, err := s.q.ListPosts(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}

	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}

	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	tagsByPost, err := s.tagsFor(ctx, ids)
	if err != nil {
		return Page{}, err
	}

	page := Page{Items: make([]ListItem, len(rows))}
	for i, r := range rows {
		if r.PublishedAt == nil {
			return Page{}, fmt.Errorf("list posts: post %s has no published_at", r.ID)
		}
		tags := tagsByPost[r.ID]
		if tags == nil {
			tags = []Tag{}
		}
		page.Items[i] = ListItem{
			Slug:        r.Slug,
			Title:       r.Title,
			Summary:     r.Summary,
			PublishedAt: *r.PublishedAt,
			Tags:        tags,
		}
	}

	if hasMore {
		last := rows[len(rows)-1]
		page.Next = &Cursor{PublishedAt: *last.PublishedAt, ID: last.ID}
	}
	return page, nil
}

// tagsFor returns each post's tags, sorted by slug. Posts without tags are
// absent from the map.
func (s *Service) tagsFor(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]Tag, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListTagsForPosts(ctx, postIDs)
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
