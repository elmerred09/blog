package querytest

import (
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

func listPostsByTagPages(t *testing.T, q *db.Queries, tagID, pageSize int32) pageFunc {
	return func(afterAt *time.Time, afterID *uuid.UUID) []listed {
		rows, err := q.ListPostsByTag(t.Context(), db.ListPostsByTagParams{
			TagID:            tagID,
			AfterPublishedAt: afterAt,
			AfterID:          afterID,
			PageSize:         pageSize,
		})
		if err != nil {
			t.Fatalf("ListPostsByTag: %v", err)
		}
		out := make([]listed, len(rows))
		for i, r := range rows {
			out[i] = listed{ID: r.ID, PublishedAt: r.PublishedAt}
		}
		return out
	}
}

func TestUpsertTag(t *testing.T) {
	t.Parallel()
	_, q := env.NewDB(t)
	ctx := t.Context()

	n, err := q.UpsertTag(ctx, db.UpsertTagParams{Slug: "postgres", Name: "Postgres"})
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if n != 1 {
		t.Errorf("first upsert: rows affected = %d, want 1 (inserted)", n)
	}

	n, err = q.UpsertTag(ctx, db.UpsertTagParams{Slug: "postgres", Name: "PostgreSQL"})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if n != 0 {
		t.Errorf("second upsert of the same slug: rows affected = %d, want 0 (skipped)", n)
	}

	got, err := q.GetTagsBySlugs(ctx, []string{"postgres"})
	if err != nil {
		t.Fatalf("GetTagsBySlugs: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Postgres" {
		t.Errorf("tags = %+v, want one tag named %q (the first import wins)", got, "Postgres")
	}
}

func TestUpsertTagRejectsUppercaseSlug(t *testing.T) {
	t.Parallel()
	_, q := env.NewDB(t)

	_, err := q.UpsertTag(t.Context(), db.UpsertTagParams{Slug: "Go", Name: "Go"})

	const checkViolation = "23514" // SQLSTATE check_violation

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("err = %v, want a Postgres error from tags_slug_check", err)
	}
	if pgErr.Code != checkViolation || pgErr.ConstraintName != "tags_slug_check" {
		t.Errorf("got SQLSTATE %s on constraint %q, want %s (check_violation) on %q",
			pgErr.Code, pgErr.ConstraintName, checkViolation, "tags_slug_check")
	}
}

func TestGetTagsBySlugs(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	for _, slug := range []string{"go", "postgres", "search"} {
		dbtest.InsertTag(t, pool, slug)
	}

	tests := []struct {
		name  string
		slugs []string
		want  []string
	}{
		{"all known, returned sorted by slug", []string{"search", "go"}, []string{"go", "search"}},
		{"unknown slugs are left out", []string{"go", "rust"}, []string{"go"}},
		{"only unknown slugs", []string{"rust"}, nil},
		{"empty input", []string{}, nil},
		{"duplicates collapse", []string{"go", "go"}, []string{"go"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := q.GetTagsBySlugs(t.Context(), tc.slugs)
			if err != nil {
				t.Fatalf("GetTagsBySlugs: %v", err)
			}
			var got []string
			for _, r := range rows {
				got = append(got, r.Slug)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("slugs = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestReplaceTagsForPost(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	post := dbtest.InsertPost(t, pool, dbtest.WithTags("old-a", "old-b"))
	other := dbtest.InsertPost(t, pool, dbtest.WithTags("old-a", "old-b"))
	oldA := dbtest.InsertTag(t, pool, "old-a")
	newA := dbtest.InsertTag(t, pool, "new-a")
	newB := dbtest.InsertTag(t, pool, "new-b")

	tagSlugs := func(t *testing.T, postID uuid.UUID) []string {
		t.Helper()
		rows, err := q.ListTagsForPost(ctx, postID)
		if err != nil {
			t.Fatalf("ListTagsForPost: %v", err)
		}
		var out []string
		for _, r := range rows {
			out = append(out, r.Slug)
		}
		return out
	}

	// replace runs the importer's link update in a transaction and checks how
	// many stale links DeleteTagsForPostExcept removed.
	replace := func(t *testing.T, tagIDs []int32, wantRemoved int64, commit bool) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		qtx := q.WithTx(tx)
		removed, err := qtx.DeleteTagsForPostExcept(ctx, db.DeleteTagsForPostExceptParams{PostID: post.ID, TagIds: tagIDs})
		if err != nil {
			t.Fatalf("DeleteTagsForPostExcept: %v", err)
		}
		if removed != wantRemoved {
			t.Errorf("DeleteTagsForPostExcept removed %d links, want %d", removed, wantRemoved)
		}
		if err := qtx.SetTagsForPost(ctx, db.SetTagsForPostParams{PostID: post.ID, TagIds: tagIDs}); err != nil {
			t.Fatalf("SetTagsForPost: %v", err)
		}
		if commit {
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("commit: %v", err)
			}
		}
	}

	t.Run("rolled back replace keeps the old tags", func(t *testing.T) {
		replace(t, []int32{newA}, 2, false)
		if got, want := tagSlugs(t, post.ID), []string{"old-a", "old-b"}; !slices.Equal(got, want) {
			t.Errorf("tags = %v, want %v", got, want)
		}
	})

	t.Run("committed replace leaves exactly the new tags", func(t *testing.T) {
		replace(t, []int32{newB, newA, newA}, 2, true) // duplicate id is ignored
		if got, want := tagSlugs(t, post.ID), []string{"new-a", "new-b"}; !slices.Equal(got, want) {
			t.Errorf("tags = %v, want %v", got, want)
		}
	})

	t.Run("same set again writes no new row versions", func(t *testing.T) {
		before := linkVersions(t, pool, post.ID)
		replace(t, []int32{newA, newB}, 0, true)
		if after := linkVersions(t, pool, post.ID); !maps.Equal(after, before) {
			t.Errorf("post_tags row versions changed:\nbefore %v\n after %v", before, after)
		}
	})

	t.Run("partial change keeps the shared link's row", func(t *testing.T) {
		before := linkVersions(t, pool, post.ID)
		replace(t, []int32{newA, oldA}, 1, true) // new-b goes, new-a stays
		if got, want := tagSlugs(t, post.ID), []string{"new-a", "old-a"}; !slices.Equal(got, want) {
			t.Errorf("tags = %v, want %v", got, want)
		}
		if after := linkVersions(t, pool, post.ID); after[newA] != before[newA] {
			t.Errorf("new-a link row version changed from %+v to %+v", before[newA], after[newA])
		}
	})

	t.Run("empty list removes every tag", func(t *testing.T) {
		replace(t, []int32{}, 2, true)
		if got := tagSlugs(t, post.ID); len(got) != 0 {
			t.Errorf("tags = %v, want none", got)
		}
	})

	t.Run("other posts' links are untouched", func(t *testing.T) {
		if got, want := tagSlugs(t, other.ID), []string{"old-a", "old-b"}; !slices.Equal(got, want) {
			t.Errorf("other post's tags = %v, want %v", got, want)
		}
	})
}

func TestListPostsByTag(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)

	tagged := tieFixture(t, pool, dbtest.WithTags("go"))
	both := dbtest.InsertPost(t, pool, dbtest.PublishedAgo(30*time.Minute), dbtest.WithTags("go", "postgres"))
	tagged = append(tagged, both)

	// Same tag, but not listable.
	dbtest.InsertPost(t, pool, dbtest.Draft(), dbtest.WithTags("go"))
	dbtest.InsertPost(t, pool, dbtest.ScheduledIn(time.Hour), dbtest.WithTags("go"))
	dbtest.InsertPost(t, pool, dbtest.Deleted(), dbtest.WithTags("go"))
	// Listable, but other tags or none.
	dbtest.InsertPost(t, pool, dbtest.WithTags("postgres"))
	dbtest.InsertPost(t, pool)

	goID := dbtest.InsertTag(t, pool, "go")
	postgresID := dbtest.InsertTag(t, pool, "postgres")
	unusedID := dbtest.InsertTag(t, pool, "unused")

	t.Run("pages through exactly the tag's listable posts", func(t *testing.T) {
		got, pages := collectPages(t, 10, listPostsByTagPages(t, q, goID, 10))
		if want := newestFirst(tagged); !slices.Equal(ids(got), want) {
			t.Errorf("rows across pages:\n got %v\nwant %v", ids(got), want)
		}
		if pages != 3 {
			t.Errorf("pages = %d, want 3 (26 posts at 10 per page)", pages)
		}
	})

	t.Run("a post with several tags appears under each", func(t *testing.T) {
		got, _ := collectPages(t, 10, listPostsByTagPages(t, q, postgresID, 10))
		if !slices.Contains(ids(got), both.ID) {
			t.Errorf("post tagged go and postgres missing from postgres listing")
		}
		if len(got) != 2 {
			t.Errorf("postgres listing has %d posts, want 2", len(got))
		}
	})

	t.Run("tag without posts", func(t *testing.T) {
		if got, _ := collectPages(t, 10, listPostsByTagPages(t, q, unusedID, 10)); len(got) != 0 {
			t.Errorf("got %d posts, want 0", len(got))
		}
	})

	t.Run("unknown tag id", func(t *testing.T) {
		if got, _ := collectPages(t, 10, listPostsByTagPages(t, q, -1, 10)); len(got) != 0 {
			t.Errorf("got %d posts, want 0", len(got))
		}
	})
}
