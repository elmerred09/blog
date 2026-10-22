// Package admin is the write side of posts: the editor's create, update, list,
// and preview. The public read API stays in internal/post.
package admin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/render"
	"github.com/elmerred09/blog/internal/slug"
	"github.com/elmerred09/blog/internal/tags"
	"github.com/elmerred09/blog/internal/validation"
)

const (
	// MaxBodyBytes is the largest markdown body a save or preview accepts.
	MaxBodyBytes = 1 << 20
	// DefaultLimit is the page size callers should use when none is given.
	DefaultLimit = 20
	// MaxLimit is the largest page size List accepts.
	MaxLimit = 50

	uniqueViolation = "23505"
)

var (
	// ErrNotFound means the post does not exist or is deleted.
	ErrNotFound = errors.New("post not found")
	// ErrConflict means the post changed since it was loaded.
	ErrConflict = errors.New("post was modified")
	// ErrSlugTaken means another post already uses this slug.
	ErrSlugTaken = errors.New("slug taken")
	// ErrInvalidLimit means ListParams.Limit is outside 1 to MaxLimit.
	ErrInvalidLimit = fmt.Errorf("limit must be between 1 and %d", MaxLimit)
)

// State is where a post is in its life, derived from published_at.
type State string

const (
	Draft     State = "draft"
	Scheduled State = "scheduled"
	Published State = "published"
)

// StateOf reports the state at now. A post whose published_at equals now is
// published, matching the public queries' published_at <= now().
func StateOf(publishedAt *time.Time, now time.Time) State {
	switch {
	case publishedAt == nil:
		return Draft
	case publishedAt.After(now):
		return Scheduled
	default:
		return Published
	}
}

// Post is a post as the editor loads it.
type Post struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Summary     string
	BodyMD      string
	PublishedAt *time.Time
	CreatedAt   time.Time
	// UpdatedAt is the optimistic-lock token. Send it back unchanged.
	UpdatedAt time.Time
	State     State
	// Tags are sorted by slug, and never nil.
	Tags []tags.Tag
}

// ListItem is one row of the admin list: enough to find a post and open it.
type ListItem struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	PublishedAt *time.Time
	UpdatedAt   time.Time
	State       State
}

// Page is one page of the admin list, newest created first.
type Page struct {
	// Items is never nil.
	Items []ListItem
	// Prev is the cursor for the newer page, or nil on the first page.
	Prev *Cursor
	// Next is the cursor for the older page, or nil on the last page.
	Next *Cursor
}

// Input is the editable fields of a post. TagNames are display names; they
// are slugified, and names that share a slug collapse to the first.
type Input struct {
	Slug        string
	Title       string
	Summary     string
	BodyMD      string
	PublishedAt *time.Time
	TagNames    []string
}

// Update is a save of an existing post. ExpectedUpdatedAt is the UpdatedAt
// the editor loaded.
type Update struct {
	ID                uuid.UUID
	ExpectedUpdatedAt time.Time
	Input
}

type ListParams struct {
	// Cursor is the previous page's Prev or Next, or nil for the first page.
	Cursor *Cursor
	// Limit is the page size, from 1 to MaxLimit.
	Limit int
}

type Service struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	render *render.Renderer
}

func NewService(pool *pgxpool.Pool, r *render.Renderer) *Service {
	return &Service{pool: pool, q: db.New(pool), render: r}
}

// List returns one page of every non-deleted post, newest created first.
func (s *Service) List(ctx context.Context, p ListParams) (Page, error) {
	if p.Limit < 1 || p.Limit > MaxLimit {
		return Page{}, fmt.Errorf("%w: got %d", ErrInvalidLimit, p.Limit)
	}
	if p.Cursor != nil && p.Cursor.Direction == Before {
		return s.listBefore(ctx, p)
	}
	return s.listAfter(ctx, p)
}

func (s *Service) listAfter(ctx context.Context, p ListParams) (Page, error) {
	params := db.ListAdminPostsParams{PageSize: int32(p.Limit + 1)}
	if p.Cursor != nil {
		params.AfterID = &p.Cursor.ID
	}
	rows, err := s.q.ListAdminPosts(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	hasMore := len(rows) > p.Limit
	if hasMore {
		rows = rows[:p.Limit]
	}
	return s.adminPage(rows, p.Cursor != nil, hasMore), nil
}

// listBefore mirrors listAfter for the newer direction. See post.Service:
// the rows come back oldest first and are reversed, and a short page means
// the top has been reached, so the first page is returned instead.
func (s *Service) listBefore(ctx context.Context, p ListParams) (Page, error) {
	rows, err := s.q.ListAdminPostsBefore(ctx, db.ListAdminPostsBeforeParams{
		BeforeID: p.Cursor.ID,
		PageSize: int32(p.Limit + 1),
	})
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	if len(rows) <= p.Limit {
		return s.listAfter(ctx, ListParams{Limit: p.Limit})
	}
	rows = rows[:p.Limit]
	slices.Reverse(rows)
	items := make([]db.ListAdminPostsRow, len(rows))
	for i, r := range rows {
		items[i] = db.ListAdminPostsRow(r)
	}
	return s.adminPage(items, true, true), nil
}

func (s *Service) adminPage(rows []db.ListAdminPostsRow, withPrev, withNext bool) Page {
	now := time.Now()
	items := make([]ListItem, len(rows))
	for i, r := range rows {
		items[i] = ListItem{
			ID:          r.ID,
			Slug:        r.Slug,
			Title:       r.Title,
			PublishedAt: r.PublishedAt,
			UpdatedAt:   r.UpdatedAt,
			State:       StateOf(r.PublishedAt, now),
		}
	}
	page := Page{Items: items}
	if withPrev && len(rows) > 0 {
		page.Prev = &Cursor{Direction: Before, ID: rows[0].ID}
	}
	if withNext && len(rows) > 0 {
		page.Next = &Cursor{Direction: After, ID: rows[len(rows)-1].ID}
	}
	return page
}

// Get returns the post, in any state, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Post, error) {
	return get(ctx, s.q, id)
}

// get reads the post with q, so a save can read it back on the same
// transaction that wrote it. That read sees the post and its tags in one
// snapshot, and the row lock from the write is held until commit, so the
// returned updated_at is the token this save produced.
func get(ctx context.Context, q *db.Queries, id uuid.UUID) (Post, error) {
	row, err := q.GetPostByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, fmt.Errorf("get post %s: %w", id, err)
	}

	tagRows, err := q.ListTagsForPost(ctx, id)
	if err != nil {
		return Post{}, fmt.Errorf("list tags for post %s: %w", id, err)
	}
	postTags := make([]tags.Tag, len(tagRows))
	for i, r := range tagRows {
		postTags[i] = tags.Tag{Slug: r.Slug, Name: r.Name}
	}

	return Post{
		ID:          row.ID,
		Slug:        row.Slug,
		Title:       row.Title,
		Summary:     row.Summary,
		BodyMD:      row.BodyMd,
		PublishedAt: row.PublishedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		State:       StateOf(row.PublishedAt, time.Now()),
		Tags:        postTags,
	}, nil
}

// Create validates and stores a post, with its tags, and returns it. The
// returned post is read inside the write's transaction, so its updated_at is
// the token this save produced.
func (s *Service) Create(ctx context.Context, in Input) (Post, error) {
	prepared, err := s.prepare(in)
	if err != nil {
		return Post{}, err
	}

	var created Post
	err = s.tx(ctx, func(q *db.Queries) error {
		row, err := q.CreatePost(ctx, db.CreatePostParams{
			Slug:        prepared.slug,
			Title:       prepared.title,
			Summary:     prepared.summary,
			BodyMd:      prepared.body,
			BodyHtml:    prepared.html,
			PublishedAt: prepared.publishedAt,
		})
		if isSlugTaken(err) {
			return fmt.Errorf("create post: %w", ErrSlugTaken)
		}
		if err != nil {
			return fmt.Errorf("create post: %w", err)
		}
		if err := syncTags(ctx, q, row.ID, prepared.tags); err != nil {
			return err
		}
		created, err = get(ctx, q, row.ID)
		return err
	})
	if err != nil {
		return Post{}, err
	}
	return created, nil
}

// Update saves a post when ExpectedUpdatedAt still matches, and returns it.
// The returned post is read inside the write's transaction, so its updated_at
// is the token this save produced. ErrConflict means another save won;
// ErrNotFound means it is gone.
func (s *Service) Update(ctx context.Context, u Update) (Post, error) {
	prepared, err := s.prepare(u.Input)
	if err != nil {
		return Post{}, err
	}

	var saved Post
	err = s.tx(ctx, func(q *db.Queries) error {
		row, err := q.UpdatePost(ctx, db.UpdatePostParams{
			ID:                u.ID,
			Slug:              prepared.slug,
			Title:             prepared.title,
			Summary:           prepared.summary,
			BodyMd:            prepared.body,
			BodyHtml:          prepared.html,
			PublishedAt:       prepared.publishedAt,
			ExpectedUpdatedAt: u.ExpectedUpdatedAt,
		})
		if isSlugTaken(err) {
			return fmt.Errorf("update post %s: %w", u.ID, ErrSlugTaken)
		}
		if err != nil {
			return fmt.Errorf("update post %s: %w", u.ID, err)
		}
		if !row.Found {
			return fmt.Errorf("update post %s: %w", u.ID, ErrNotFound)
		}
		if row.UpdatedAt == nil {
			return fmt.Errorf("update post %s: %w", u.ID, ErrConflict)
		}
		if err := syncTags(ctx, q, u.ID, prepared.tags); err != nil {
			return err
		}
		saved, err = get(ctx, q, u.ID)
		return err
	})
	if err != nil {
		return Post{}, err
	}
	return saved, nil
}

// Preview renders markdown the way a save would store it. An oversized body
// is a *validation.Error.
func (s *Service) Preview(md string) (string, error) {
	if len(md) > MaxBodyBytes {
		return "", validation.New(validation.Problem{Field: "body_md", Message: bodyTooBig(len(md))})
	}
	html, err := s.render.Render([]byte(md))
	if err != nil {
		return "", fmt.Errorf("preview: %w", err)
	}
	return html, nil
}

// prepared is an Input that passed validation, plus its rendered HTML.
type prepared struct {
	slug        string
	title       string
	summary     string
	body        string
	html        string
	publishedAt *time.Time
	tags        []tags.Tag
}

func (s *Service) prepare(in Input) (prepared, error) {
	p, err := validate(in)
	if err != nil {
		return prepared{}, err
	}
	html, err := s.render.Render([]byte(p.body))
	if err != nil {
		return prepared{}, fmt.Errorf("render body: %w", err)
	}
	p.html = html
	return p, nil
}

// validate checks the input and returns a *validation.Error listing every
// problem, or the cleaned fields.
func validate(in Input) (prepared, error) {
	var problems []validation.Problem

	title := strings.TrimSpace(in.Title)
	if title == "" {
		problems = append(problems, validation.Problem{Field: "title", Message: "title is required"})
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		problems = append(problems, validation.Problem{Field: "summary", Message: "summary is required"})
	}
	if !slug.Valid(in.Slug) {
		problems = append(problems, validation.Problem{
			Field:   "slug",
			Message: fmt.Sprintf("slug %q must be lowercase letters, digits, and single dashes", in.Slug),
		})
	}
	if len(in.BodyMD) > MaxBodyBytes {
		problems = append(problems, validation.Problem{Field: "body_md", Message: bodyTooBig(len(in.BodyMD))})
	}

	postTags, err := tags.FromNames(in.TagNames)
	if err != nil {
		for _, tagErr := range flatten(err) {
			problems = append(problems, validation.Problem{Field: "tags", Message: tagErr.Error()})
		}
	}

	if err := validation.New(problems...); err != nil {
		return prepared{}, err
	}
	if postTags == nil {
		postTags = []tags.Tag{}
	}
	return prepared{
		slug:        in.Slug,
		title:       title,
		summary:     summary,
		body:        in.BodyMD,
		publishedAt: in.PublishedAt,
		tags:        postTags,
	}, nil
}

func bodyTooBig(n int) string {
	return fmt.Sprintf("body is %d bytes, the limit is %d", n, MaxBodyBytes)
}

// flatten reports every error errors.Join bundled, or err itself.
func flatten(err error) []error {
	type joined interface{ Unwrap() []error }
	if u, ok := err.(joined); ok {
		return u.Unwrap()
	}
	return []error{err}
}

func syncTags(ctx context.Context, q *db.Queries, postID uuid.UUID, postTags []tags.Tag) error {
	ids, _, err := tags.Ensure(ctx, q, postTags)
	if err != nil {
		return err
	}
	tagIDs := make([]int32, len(postTags))
	for i, t := range postTags {
		id, ok := ids[t.Slug]
		if !ok {
			return fmt.Errorf("tag %q was not stored", t.Slug)
		}
		tagIDs[i] = id
	}
	if _, _, err := tags.Link(ctx, q, postID, tagIDs); err != nil {
		return fmt.Errorf("post %s: %w", postID, err)
	}
	return nil
}

func (s *Service) tx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(db.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func isSlugTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == "posts_slug_key"
}
