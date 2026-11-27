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
	page, err := s.list(ctx, keyset{
		after: func(ctx context.Context, at *time.Time, id *uuid.UUID, n int32) ([]Row, error) {
			rows, err := s.q.ListPosts(ctx, db.ListPostsParams{AfterPublishedAt: at, AfterID: id, PageSize: n})
			return convertRows(rows, err)
		},
		before: func(ctx context.Context, at time.Time, id uuid.UUID, n int32) ([]Row, error) {
			rows, err := s.q.ListPostsBefore(ctx, db.ListPostsBeforeParams{BeforePublishedAt: at, BeforeID: id, PageSize: n})
			return convertRows(rows, err)
		},
	}, p)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	return page, nil
}

// TagPage is one page of a tag's published posts.
type TagPage struct {
	Tag Tag
	Page
}

// ListByTag returns one page of the published posts with this tag, newest
// first, with all of their tags. An empty page is ErrNotFound: an unknown tag,
// a tag on no published post (so a draft's tag name can't leak, as in the
// public tag list), or a cursor past the end.
func (s *Service) ListByTag(ctx context.Context, tagSlug string, p ListParams) (TagPage, error) {
	if p.Limit < 1 || p.Limit > MaxLimit {
		return TagPage{}, fmt.Errorf("%w: got %d", ErrInvalidLimit, p.Limit)
	}
	found, err := s.q.GetTagsBySlugs(ctx, []string{tagSlug})
	if err != nil {
		return TagPage{}, fmt.Errorf("get tag %q: %w", tagSlug, err)
	}
	if len(found) == 0 {
		return TagPage{}, ErrNotFound
	}
	tag := found[0]

	page, err := s.list(ctx, keyset{
		after: func(ctx context.Context, at *time.Time, id *uuid.UUID, n int32) ([]Row, error) {
			rows, err := s.q.ListPostsByTag(ctx, db.ListPostsByTagParams{TagID: tag.ID, AfterPublishedAt: at, AfterID: id, PageSize: n})
			return convertRows(rows, err)
		},
		before: func(ctx context.Context, at time.Time, id uuid.UUID, n int32) ([]Row, error) {
			rows, err := s.q.ListPostsByTagBefore(ctx, db.ListPostsByTagBeforeParams{TagID: tag.ID, BeforePublishedAt: at, BeforeID: id, PageSize: n})
			return convertRows(rows, err)
		},
	}, p)
	if err != nil {
		return TagPage{}, fmt.Errorf("list posts tagged %q: %w", tagSlug, err)
	}
	if len(page.Items) == 0 {
		return TagPage{}, ErrNotFound
	}
	return TagPage{Tag: Tag{Slug: tag.Slug, Name: tag.Name, Color: tag.Color}, Page: page}, nil
}

// keyset is a list's two queries on the (published_at, id) order. after reads
// older rows than the cursor (nil for the newest), newest first; before reads
// newer rows than the cursor, oldest first. Each returns at most n rows.
type keyset struct {
	after  func(ctx context.Context, at *time.Time, id *uuid.UUID, n int32) ([]Row, error)
	before func(ctx context.Context, at time.Time, id uuid.UUID, n int32) ([]Row, error)
}

func (s *Service) list(ctx context.Context, k keyset, p ListParams) (Page, error) {
	if p.Limit < 1 || p.Limit > MaxLimit {
		return Page{}, fmt.Errorf("%w: got %d", ErrInvalidLimit, p.Limit)
	}
	if p.Cursor != nil && p.Cursor.Direction == Before {
		return s.listBefore(ctx, k, p)
	}
	return s.listAfter(ctx, k, p)
}

// listAfter is a page of posts older than the cursor (or the newest posts,
// when there is no cursor), read in the list order.
func (s *Service) listAfter(ctx context.Context, k keyset, p ListParams) (Page, error) {
	var (
		at *time.Time
		id *uuid.UUID
	)
	if p.Cursor != nil {
		at, id = &p.Cursor.PublishedAt, &p.Cursor.ID
	}
	// One extra row says whether another page exists, without a count query.
	rows, err := k.after(ctx, at, id, int32(p.Limit+1))
	if err != nil {
		return Page{}, err
	}

	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}
	// A cursor was given, so something newer than this page exists: the row
	// the cursor itself points at.
	return s.pageFrom(ctx, rows, p.Cursor != nil, hasMore)
}

// listBefore is a page of posts newer than the cursor. The query returns
// them oldest first; they're reversed so the page reads newest first, like
// every other page. A short result has reached the top, and the first page
// replaces it, so arriving there shows exactly what no cursor shows.
func (s *Service) listBefore(ctx context.Context, k keyset, p ListParams) (Page, error) {
	rows, err := k.before(ctx, p.Cursor.PublishedAt, p.Cursor.ID, int32(p.Limit+1))
	if err != nil {
		return Page{}, err
	}
	if len(rows) <= p.Limit {
		return s.listAfter(ctx, k, ListParams{Limit: p.Limit})
	}
	rows = rows[:p.Limit]
	slices.Reverse(rows)
	return s.pageFrom(ctx, rows, true, true)
}

// pageFrom attaches tags and the cursors. withPrev and withNext say whether
// each direction has another page; the cursors point at this page's ends.
func (s *Service) pageFrom(ctx context.Context, rows []Row, withPrev, withNext bool) (Page, error) {
	items, err := ListItems(ctx, s.q, rows)
	if err != nil {
		return Page{}, err
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

// convertRows turns a list query's rows, which all select Row's columns in
// Row's order, into Rows.
func convertRows[T ~struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Summary     string
	PublishedAt *time.Time
	WordCount   int32
}](rows []T, err error) ([]Row, error) {
	if err != nil {
		return nil, err
	}
	out := make([]Row, len(rows))
	for i, r := range rows {
		out[i] = Row(r)
	}
	return out, nil
}

// Row is the columns a list-style query (ListPosts, ListPostsByTag,
// SearchPosts) selects for
// each post.
type Row struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Summary     string
	PublishedAt *time.Time
	WordCount   int32
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
			Slug:           r.Slug,
			Title:          r.Title,
			Summary:        r.Summary,
			PublishedAt:    *r.PublishedAt,
			ReadingMinutes: ReadingMinutes(r.WordCount),
			Tags:           tags,
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
		tagsByPost[r.PostID] = append(tagsByPost[r.PostID], Tag{Slug: r.Slug, Name: r.Name, Color: r.Color})
	}
	return tagsByPost, nil
}

// SitemapEntry is one published post in the sitemap.
type SitemapEntry struct {
	Slug         string
	LastModified time.Time
}

// SitemapEntries returns every published post, newest published first, up to
// one sitemap file. Drafts, scheduled posts, and deleted posts are absent.
// LastModified is the later of updated_at and published_at: the moment the
// post last changed for a reader.
func (s *Service) SitemapEntries(ctx context.Context) ([]SitemapEntry, error) {
	rows, err := s.q.ListSitemapPosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sitemap posts: %w", err)
	}
	entries := make([]SitemapEntry, len(rows))
	for i, r := range rows {
		entries[i] = SitemapEntry{Slug: r.Slug, LastModified: r.LastModified}
	}
	return entries, nil
}

// Adjacent returns the published posts just newer and just older than the
// published post with this slug, or ErrNotFound.
func (s *Service) Adjacent(ctx context.Context, slug string) (Adjacent, error) {
	row, err := s.q.GetAdjacentPosts(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Adjacent{}, ErrNotFound
	}
	if err != nil {
		return Adjacent{}, fmt.Errorf("get posts adjacent to %q: %w", slug, err)
	}
	return Adjacent{
		Newer: neighbor(row.NewerSlug, row.NewerTitle),
		Older: neighbor(row.OlderSlug, row.OlderTitle),
	}, nil
}

func neighbor(slug, title *string) *Neighbor {
	if slug == nil || title == nil {
		return nil
	}
	return &Neighbor{Slug: *slug, Title: *title}
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
		tags[i] = Tag{Slug: r.Slug, Name: r.Name, Color: r.Color}
	}

	return Post{
		ListItem: ListItem{
			Slug:           row.Slug,
			Title:          row.Title,
			Summary:        row.Summary,
			PublishedAt:    *row.PublishedAt,
			ReadingMinutes: ReadingMinutes(row.WordCount),
			Tags:           tags,
		},
		BodyHTML:  row.BodyHtml,
		UpdatedAt: row.UpdatedAt,
	}, nil
}
