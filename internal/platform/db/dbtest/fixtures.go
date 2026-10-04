package dbtest

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/db"
)

// Fixtures insert rows with plain SQL instead of the generated queries, so a
// bug in a query under test can't hide behind the fixture that set it up.
//
// Times are relative to the database clock (now()), not time.Now(), because
// the queries compare against now() and the two clocks differ slightly.

// Post is a row inserted by InsertPost.
type Post struct {
	ID          uuid.UUID
	Slug        string
	PublishedAt *time.Time
}

// PostOption customizes a post inserted by InsertPost.
type PostOption func(*postSpec)

type postSpec struct {
	slug    string
	title   string
	exactAt *time.Time     // published_at = exactAt
	offset  *time.Duration // published_at = now() + offset
	deleted bool
	tags    []string
}

// WithSlug sets the slug. The default is unique per call.
func WithSlug(slug string) PostOption {
	return func(s *postSpec) { s.slug = slug }
}

// WithTitle sets the title. The default is derived from the slug.
func WithTitle(title string) PostOption {
	return func(s *postSpec) { s.title = title }
}

// PublishedAgo publishes the post at now() - ago. The default is one hour ago.
func PublishedAgo(ago time.Duration) PostOption {
	return func(s *postSpec) {
		d := -ago
		s.offset, s.exactAt = &d, nil
	}
}

// PublishedAt publishes the post at exactly t. Use it with Now to create posts
// with identical published_at values (ties).
func PublishedAt(t time.Time) PostOption {
	return func(s *postSpec) { s.exactAt, s.offset = &t, nil }
}

// ScheduledIn publishes the post at now() + in, so it isn't listable yet.
func ScheduledIn(in time.Duration) PostOption {
	return func(s *postSpec) { s.offset, s.exactAt = &in, nil }
}

// Draft leaves published_at NULL.
func Draft() PostOption {
	return func(s *postSpec) { s.offset, s.exactAt = nil, nil }
}

// Deleted soft-deletes the post (deleted_at = now()).
func Deleted() PostOption {
	return func(s *postSpec) { s.deleted = true }
}

// WithTags links the post to tags by slug, creating missing tags with
// InsertTag.
func WithTags(slugs ...string) PostOption {
	return func(s *postSpec) { s.tags = append(s.tags, slugs...) }
}

// InsertPost inserts a post, published one hour ago unless options say
// otherwise, and links any tags.
func InsertPost(t testing.TB, conn db.DBTX, opts ...PostOption) Post {
	t.Helper()

	hourAgo := -time.Hour
	spec := postSpec{
		slug:   "post-" + uuid.NewString()[:8],
		offset: &hourAgo,
	}
	for _, opt := range opts {
		opt(&spec)
	}
	if spec.title == "" {
		spec.title = "Title of " + spec.slug
	}

	var offsetSecs *float64
	if spec.offset != nil {
		secs := spec.offset.Seconds()
		offsetSecs = &secs
	}

	const q = `
INSERT INTO posts (slug, title, summary, body_md, body_html, published_at, deleted_at)
VALUES (
    $1, $2, 'Summary of ' || $1, '# ' || $2, '<h1>' || $2 || '</h1>',
    coalesce($3::timestamptz, now() + $4::float8 * interval '1 second'),
    CASE WHEN $5::bool THEN now() END
)
RETURNING id, slug, published_at`

	var p Post
	err := conn.QueryRow(t.Context(), q, spec.slug, spec.title, spec.exactAt, offsetSecs, spec.deleted).
		Scan(&p.ID, &p.Slug, &p.PublishedAt)
	if err != nil {
		t.Fatalf("insert post %q: %v", spec.slug, err)
	}

	for _, tag := range spec.tags {
		TagPost(t, conn, p.ID, InsertTag(t, conn, tag))
	}
	return p
}

// InsertTag returns the id of the tag with this slug, creating it if needed.
// The name is set to the slug.
func InsertTag(t testing.TB, conn db.DBTX, slug string) int32 {
	t.Helper()

	const q = `
WITH ins AS (
    INSERT INTO tags (slug, name) VALUES ($1, $1)
    ON CONFLICT (slug) DO NOTHING
    RETURNING id
)
SELECT id FROM ins
UNION ALL
SELECT id FROM tags WHERE slug = $1
LIMIT 1`

	var id int32
	if err := conn.QueryRow(t.Context(), q, slug).Scan(&id); err != nil {
		t.Fatalf("insert tag %q: %v", slug, err)
	}
	return id
}

// TagPost links a post to a tag.
func TagPost(t testing.TB, conn db.DBTX, postID uuid.UUID, tagID int32) {
	t.Helper()

	const q = `INSERT INTO post_tags (post_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	if _, err := conn.Exec(t.Context(), q, postID, tagID); err != nil {
		t.Fatalf("tag post %s with %d: %v", postID, tagID, err)
	}
}

// Now returns the database's current time, for building exact published_at
// values (for example ties) that stay consistent with now() in queries.
func Now(t testing.TB, conn db.DBTX) time.Time {
	t.Helper()

	var now time.Time
	if err := conn.QueryRow(t.Context(), `SELECT now()`).Scan(&now); err != nil {
		t.Fatalf("select now(): %v", err)
	}
	return now
}
