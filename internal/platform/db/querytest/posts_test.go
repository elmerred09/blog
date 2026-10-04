package querytest

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

func listPostsPages(t *testing.T, q *db.Queries, pageSize int32) pageFunc {
	return func(afterAt *time.Time, afterID *uuid.UUID) []listed {
		rows, err := q.ListPosts(t.Context(), db.ListPostsParams{
			AfterPublishedAt: afterAt,
			AfterID:          afterID,
			PageSize:         pageSize,
		})
		if err != nil {
			t.Fatalf("ListPosts: %v", err)
		}
		out := make([]listed, len(rows))
		for i, r := range rows {
			out[i] = listed{ID: r.ID, PublishedAt: r.PublishedAt}
		}
		return out
	}
}

func TestListPostsVisibility(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	published := dbtest.InsertPost(t, pool)
	dbtest.InsertPost(t, pool, dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.Deleted())
	dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Hour), dbtest.Deleted())

	got, _ := collectPages(t, 10, listPostsPages(t, q, 10))
	if want := []uuid.UUID{published.ID}; !slices.Equal(ids(got), want) {
		t.Errorf("ListPosts = %v, want only the published post %v", ids(got), want)
	}
}

func TestListPostsPagination(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	posts := tieFixture(t, pool)
	dbtest.InsertPost(t, pool, dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.ScheduledIn(time.Hour))

	tests := []struct {
		name      string
		pageSize  int32
		wantPages int
	}{
		{"tie group straddles a page boundary", 10, 3},
		{"page size divides the total evenly", 5, 6}, // 5 full pages, then an empty one
		{"single page", 100, 1},
		{"one row per page", 1, 26},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, pages := collectPages(t, int(tc.pageSize), listPostsPages(t, q, tc.pageSize))
			if want := newestFirst(posts); !slices.Equal(ids(got), want) {
				t.Errorf("rows across pages:\n got %v\nwant %v", ids(got), want)
			}
			if pages != tc.wantPages {
				t.Errorf("pages = %d, want %d", pages, tc.wantPages)
			}
		})
	}
}

func TestGetPostBySlug(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	published := dbtest.InsertPost(t, pool, dbtest.WithSlug("published"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), dbtest.Deleted())

	tests := []struct {
		slug     string
		wantID   uuid.UUID
		notFound bool
	}{
		{slug: "published", wantID: published.ID},
		{slug: "draft", notFound: true},
		{slug: "scheduled", notFound: true},
		{slug: "deleted", notFound: true},
		{slug: "missing", notFound: true},
	}
	for _, tc := range tests {
		t.Run(tc.slug, func(t *testing.T) {
			got, err := q.GetPostBySlug(t.Context(), tc.slug)
			if tc.notFound {
				if !errors.Is(err, pgx.ErrNoRows) {
					t.Fatalf("err = %v, want pgx.ErrNoRows", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetPostBySlug: %v", err)
			}
			if got.ID != tc.wantID {
				t.Errorf("id = %v, want %v", got.ID, tc.wantID)
			}
		})
	}
}

func TestUpsertPost(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	publishedAt := dbtest.Now(t, pool).Add(-time.Hour).Truncate(time.Microsecond)
	params := db.UpsertPostParams{
		Slug:        "hello",
		Title:       "Hello",
		Summary:     "First",
		BodyMd:      "# Hello",
		BodyHtml:    "<h1>Hello</h1>",
		PublishedAt: &publishedAt,
	}

	id, err := q.UpsertPost(ctx, params)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	t.Run("re-import updates fields and keeps id", func(t *testing.T) {
		edited := params
		edited.Title, edited.BodyMd, edited.BodyHtml = "Hello again", "# Edited", "<h1>Edited</h1>"

		id2, err := q.UpsertPost(ctx, edited)
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if id2 != id {
			t.Errorf("id changed from %v to %v", id, id2)
		}
		got, err := q.GetPostBySlug(ctx, "hello")
		if err != nil {
			t.Fatalf("GetPostBySlug: %v", err)
		}
		if got.Title != edited.Title || got.BodyMd != edited.BodyMd || got.BodyHtml != edited.BodyHtml {
			t.Errorf("got title %q body %q html %q, want the edited values", got.Title, got.BodyMd, got.BodyHtml)
		}
	})

	t.Run("re-import restores a soft-deleted post", func(t *testing.T) {
		if _, err := q.SoftDeletePost(ctx, id); err != nil {
			t.Fatalf("SoftDeletePost: %v", err)
		}
		if _, err := q.GetPostBySlug(ctx, "hello"); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("deleted post still visible: err = %v", err)
		}
		if _, err := q.UpsertPost(ctx, params); err != nil {
			t.Fatalf("re-import: %v", err)
		}
		if _, err := q.GetPostBySlug(ctx, "hello"); err != nil {
			t.Errorf("post not restored: %v", err)
		}
	})

	t.Run("search vector follows the content", func(t *testing.T) {
		edited := params
		edited.Title = "Keyset pagination"
		if _, err := q.UpsertPost(ctx, edited); err != nil {
			t.Fatalf("update: %v", err)
		}
		var matches bool
		err := pool.QueryRow(ctx,
			`SELECT search_vector @@ websearch_to_tsquery('english', 'keyset') FROM posts WHERE id = $1`, id,
		).Scan(&matches)
		if err != nil {
			t.Fatalf("query search_vector: %v", err)
		}
		if !matches {
			t.Error("search_vector doesn't match the new title")
		}
	})
}

func TestSoftDeletePost(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	p := dbtest.InsertPost(t, pool)

	got, err := q.SoftDeletePost(ctx, p.ID)
	if err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if got != p.ID {
		t.Errorf("returned id %v, want %v", got, p.ID)
	}

	if _, err := q.SoftDeletePost(ctx, p.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("second delete: err = %v, want pgx.ErrNoRows", err)
	}
	if _, err := q.SoftDeletePost(ctx, uuid.New()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("unknown id: err = %v, want pgx.ErrNoRows", err)
	}

	rows, _ := collectPages(t, 10, listPostsPages(t, q, 10))
	if len(rows) != 0 {
		t.Errorf("ListPosts returned %d rows after delete, want 0", len(rows))
	}
}
