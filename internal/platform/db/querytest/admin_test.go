package querytest

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

const uniqueViolation = "23505" // SQLSTATE unique_violation

func wantUniqueViolation(t *testing.T, err error, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("err = %v, want a Postgres error from %s", err, constraint)
	}
	if pgErr.Code != uniqueViolation || pgErr.ConstraintName != constraint {
		t.Errorf("got SQLSTATE %s on constraint %q, want %s (unique_violation) on %q",
			pgErr.Code, pgErr.ConstraintName, uniqueViolation, constraint)
	}
}

func getPost(t *testing.T, q *db.Queries, id uuid.UUID) db.GetPostByIDRow {
	t.Helper()
	p, err := q.GetPostByID(t.Context(), id)
	if err != nil {
		t.Fatalf("GetPostByID(%s): %v", id, err)
	}
	return p
}

func updateParams(id uuid.UUID, slug string, expected time.Time) db.UpdatePostParams {
	return db.UpdatePostParams{
		ID:                id,
		Slug:              slug,
		Title:             "Title of " + slug,
		Summary:           "Summary of " + slug,
		BodyMd:            "# " + slug,
		BodyHtml:          "<h1>" + slug + "</h1>",
		ExpectedUpdatedAt: expected,
	}
}

func TestCreatePost(t *testing.T) {
	t.Parallel()
	_, q := env.NewDB(t)
	ctx := t.Context()

	params := db.CreatePostParams{
		Slug:     "hello",
		Title:    "Hello",
		Summary:  "A first post.",
		BodyMd:   "# Hello",
		BodyHtml: "<h1>Hello</h1>",
	}
	created, err := q.CreatePost(ctx, params)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	got := getPost(t, q, created.ID)
	if got.Slug != params.Slug || got.Title != params.Title || got.Summary != params.Summary || got.BodyMd != params.BodyMd {
		t.Errorf("stored %+v, want the created fields %+v", got, params)
	}
	if got.PublishedAt != nil {
		t.Errorf("published_at = %v, want NULL (draft)", got.PublishedAt)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) {
		t.Errorf("returned updated_at %v, stored %v; want them equal (it's the lock token)",
			created.UpdatedAt, got.UpdatedAt)
	}

	_, err = q.CreatePost(ctx, params)
	wantUniqueViolation(t, err, "posts_slug_key")
}

func TestUpdatePost(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	p := dbtest.InsertPost(t, pool, dbtest.Draft())
	loaded := getPost(t, q, p.ID).UpdatedAt

	publishAt := dbtest.Now(t, pool).Add(-time.Minute).Truncate(time.Microsecond)
	first := updateParams(p.ID, "renamed", loaded)
	first.PublishedAt = &publishAt
	res, err := q.UpdatePost(ctx, first)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	if !res.Found || res.UpdatedAt == nil {
		t.Fatalf("first save = %+v, want found with a new updated_at", res)
	}
	if !res.UpdatedAt.After(loaded) {
		t.Errorf("new updated_at %v, want after the loaded %v", *res.UpdatedAt, loaded)
	}
	got := getPost(t, q, p.ID)
	if got.Slug != "renamed" || got.Title != first.Title || got.BodyMd != first.BodyMd {
		t.Errorf("stored %+v, want the saved fields", got)
	}
	if got.PublishedAt == nil || !got.PublishedAt.Equal(publishAt) {
		t.Errorf("published_at = %v, want %v", got.PublishedAt, publishAt)
	}
	if !got.UpdatedAt.Equal(*res.UpdatedAt) {
		t.Errorf("stored updated_at %v, want the returned %v", got.UpdatedAt, *res.UpdatedAt)
	}
	token := *res.UpdatedAt

	t.Run("stale token conflicts and writes nothing", func(t *testing.T) {
		before := postVersion(t, pool, "renamed")
		res, err := q.UpdatePost(ctx, updateParams(p.ID, "stale", loaded))
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if !res.Found || res.UpdatedAt != nil {
			t.Errorf("result = %+v, want found with no updated_at (conflict)", res)
		}
		if after := postVersion(t, pool, "renamed"); after != before {
			t.Errorf("row version changed from %v to %v, want no write", before, after)
		}
	})

	t.Run("an unchanged save still issues a new token", func(t *testing.T) {
		same := updateParams(p.ID, "renamed", token)
		same.PublishedAt = &publishAt
		res, err := q.UpdatePost(ctx, same)
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if !res.Found || res.UpdatedAt == nil || !res.UpdatedAt.After(token) {
			t.Fatalf("result = %+v, want found with updated_at after %v", res, token)
		}
		token = *res.UpdatedAt
	})

	t.Run("clearing published_at makes a draft again", func(t *testing.T) {
		res, err := q.UpdatePost(ctx, updateParams(p.ID, "renamed", token))
		if err != nil {
			t.Fatalf("UpdatePost: %v", err)
		}
		if !res.Found || res.UpdatedAt == nil {
			t.Fatalf("result = %+v, want found with a new updated_at", res)
		}
		if got := getPost(t, q, p.ID); got.PublishedAt != nil {
			t.Errorf("published_at = %v, want NULL", got.PublishedAt)
		}
		token = *res.UpdatedAt
	})

	t.Run("taken slug", func(t *testing.T) {
		other := dbtest.InsertPost(t, pool)
		_, err := q.UpdatePost(ctx, updateParams(p.ID, other.Slug, token))
		wantUniqueViolation(t, err, "posts_slug_key")
	})
}

func TestUpdatePostNotFound(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	deleted := dbtest.InsertPost(t, pool, dbtest.Deleted())
	var deletedAt time.Time
	if err := pool.QueryRow(t.Context(), `SELECT updated_at FROM posts WHERE id = $1`, deleted.ID).Scan(&deletedAt); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}

	tests := []struct {
		name     string
		id       uuid.UUID
		expected time.Time
	}{
		{"unknown id", uuid.New(), time.Now()},
		// The right token, so only deleted_at can be what rejects it.
		{"deleted post", deleted.ID, deletedAt},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := q.UpdatePost(t.Context(), updateParams(tc.id, "whatever", tc.expected))
			if err != nil {
				t.Fatalf("UpdatePost: %v", err)
			}
			if res.Found || res.UpdatedAt != nil {
				t.Errorf("result = %+v, want not found", res)
			}
		})
	}
}

// Two saves from the same loaded version race. The second blocks on the first
// one's row lock; once the first commits, Postgres re-checks the second's
// WHERE against the new row version, so it must report a conflict rather than
// overwrite the first save.
func TestUpdatePostConcurrentSaves(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	p := dbtest.InsertPost(t, pool)
	loaded := getPost(t, q, p.ID).UpdatedAt

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	first, err := q.WithTx(tx).UpdatePost(ctx, updateParams(p.ID, "first", loaded))
	if err != nil || !first.Found || first.UpdatedAt == nil {
		t.Fatalf("first save = %+v, %v; want success", first, err)
	}

	type result struct {
		row db.UpdatePostRow
		err error
	}
	second := make(chan result, 1)
	go func() {
		row, err := q.UpdatePost(ctx, updateParams(p.ID, "second", loaded))
		second <- result{row, err}
	}()

	waitForLockWait(t, pool)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit first save: %v", err)
	}

	r := <-second
	if r.err != nil {
		t.Fatalf("second save: %v", r.err)
	}
	if !r.row.Found || r.row.UpdatedAt != nil {
		t.Errorf("second save = %+v, want found with no updated_at (conflict)", r.row)
	}
	if got := getPost(t, q, p.ID); got.Slug != "first" {
		t.Errorf("slug = %q, want %q (the first save kept)", got.Slug, "first")
	}
}

// waitForLockWait polls until a backend in this database is blocked on a lock.
func waitForLockWait(t *testing.T, conn db.DBTX) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		err := conn.QueryRow(t.Context(), `
SELECT EXISTS (
    SELECT 1 FROM pg_stat_activity
    WHERE datname = current_database() AND wait_event_type = 'Lock'
)`).Scan(&waiting)
		if err != nil {
			t.Fatalf("read pg_stat_activity: %v", err)
		}
		if waiting {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no backend started waiting on a lock")
}

func TestGetPostByID(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	tests := []struct {
		name  string
		opts  []dbtest.PostOption
		found bool
	}{
		{"published", nil, true},
		{"draft", []dbtest.PostOption{dbtest.Draft()}, true},
		{"scheduled", []dbtest.PostOption{dbtest.ScheduledIn(time.Hour)}, true},
		{"deleted", []dbtest.PostOption{dbtest.Deleted()}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const body = "# Markdown source"
			p := dbtest.InsertPost(t, pool, append(tc.opts, dbtest.WithBody(body))...)

			got, err := q.GetPostByID(t.Context(), p.ID)
			if !tc.found {
				if !errors.Is(err, pgx.ErrNoRows) {
					t.Errorf("err = %v, want pgx.ErrNoRows", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetPostByID: %v", err)
			}
			if got.Slug != p.Slug || got.BodyMd != body {
				t.Errorf("got slug %q, body %q; want %q, %q", got.Slug, got.BodyMd, p.Slug, body)
			}
		})
	}

	t.Run("unknown id", func(t *testing.T) {
		if _, err := q.GetPostByID(t.Context(), uuid.New()); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("err = %v, want pgx.ErrNoRows", err)
		}
	})
}

func TestListAdminPosts(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	var want []uuid.UUID
	for i := range 12 {
		var opt dbtest.PostOption
		switch i % 3 {
		case 0:
			opt = dbtest.PublishedAgo(time.Duration(i) * time.Hour)
		case 1:
			opt = dbtest.Draft()
		case 2:
			opt = dbtest.ScheduledIn(time.Duration(i) * time.Hour)
		}
		want = append(want, dbtest.InsertPost(t, pool, opt).ID)
	}
	dbtest.InsertPost(t, pool, dbtest.Deleted())
	slices.SortFunc(want, func(a, b uuid.UUID) int { return bytes.Compare(b[:], a[:]) })

	const pageSize = 5
	rows, pages := collectPages(t, pageSize, func(_ *time.Time, afterID *uuid.UUID) []listed {
		got, err := q.ListAdminPosts(t.Context(), db.ListAdminPostsParams{AfterID: afterID, PageSize: pageSize})
		if err != nil {
			t.Fatalf("ListAdminPosts: %v", err)
		}
		out := make([]listed, len(got))
		for i, r := range got {
			out[i] = listed{ID: r.ID, PublishedAt: r.PublishedAt}
		}
		return out
	})

	if got := ids(rows); !slices.Equal(got, want) {
		t.Errorf("ids = %v,\nwant every non-deleted post by id descending: %v", got, want)
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3 (5 + 5 + 2)", pages)
	}
}
