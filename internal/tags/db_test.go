package tags_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/tags"
)

func TestEnsure(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	goID := dbtest.InsertTag(t, pool, "go") // named "go" by the fixture

	ids, created, err := tags.Ensure(ctx, q, toTags([]tag{
		{"postgres", "Postgres"}, {"go", "Go"}, {"postgres", "POSTGRES"},
	}))
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if created != 1 {
		t.Errorf("created = %d, want 1 (only postgres was missing)", created)
	}
	if ids["go"] != goID {
		t.Errorf("ids[go] = %d, want the existing tag's id %d", ids["go"], goID)
	}
	if len(ids) != 2 {
		t.Errorf("ids = %v, want go and postgres", ids)
	}

	rows, err := q.GetTagsBySlugs(ctx, []string{"go", "postgres"})
	if err != nil {
		t.Fatalf("GetTagsBySlugs: %v", err)
	}
	names := map[string]string{}
	for _, r := range rows {
		names[r.Slug] = r.Name
		if ids[r.Slug] != r.ID {
			t.Errorf("ids[%s] = %d, want %d", r.Slug, ids[r.Slug], r.ID)
		}
	}
	if want := map[string]string{"go": "go", "postgres": "Postgres"}; !maps.Equal(names, want) {
		t.Errorf("names = %v, want %v (existing tags keep their name, a new one takes the first)", names, want)
	}

	t.Run("no tags", func(t *testing.T) {
		ids, created, err := tags.Ensure(ctx, q, nil)
		if err != nil || created != 0 || len(ids) != 0 {
			t.Errorf("Ensure(nil) = %v, %d, %v; want empty, 0, nil", ids, created, err)
		}
	})
}

func TestLink(t *testing.T) {
	t.Parallel()
	pool, q := env.NewDB(t)
	ctx := t.Context()

	post := dbtest.InsertPost(t, pool, dbtest.WithTags("old-a", "old-b"))
	other := dbtest.InsertPost(t, pool, dbtest.WithTags("old-a", "old-b"))
	oldA := dbtest.InsertTag(t, pool, "old-a")
	newA := dbtest.InsertTag(t, pool, "new-a")
	newB := dbtest.InsertTag(t, pool, "new-b")

	link := func(t *testing.T, tagIDs []int32, wantAdded, wantRemoved int) {
		t.Helper()
		added, removed, err := tags.Link(ctx, q, post.ID, tagIDs)
		if err != nil {
			t.Fatalf("Link(%v): %v", tagIDs, err)
		}
		if added != wantAdded || removed != wantRemoved {
			t.Errorf("Link(%v) added %d, removed %d; want %d, %d", tagIDs, added, removed, wantAdded, wantRemoved)
		}
	}

	t.Run("replaces the set", func(t *testing.T) {
		link(t, []int32{newB, newA, newA}, 2, 2) // a duplicate id is ignored
		if got, want := tagSlugs(t, q, post.ID), []string{"new-a", "new-b"}; !slices.Equal(got, want) {
			t.Errorf("tags = %v, want %v", got, want)
		}
	})

	t.Run("same set writes no new row versions", func(t *testing.T) {
		before := linkVersions(t, pool, post.ID)
		link(t, []int32{newA, newB}, 0, 0)
		if after := linkVersions(t, pool, post.ID); !maps.Equal(after, before) {
			t.Errorf("post_tags row versions changed:\nbefore %v\n after %v", before, after)
		}
	})

	t.Run("partial change keeps the shared link's row", func(t *testing.T) {
		before := linkVersions(t, pool, post.ID)
		link(t, []int32{newA, oldA}, 1, 1) // new-b goes, new-a stays
		after := linkVersions(t, pool, post.ID)
		if after[newA] != before[newA] {
			t.Errorf("new-a link was rewritten: %v -> %v", before[newA], after[newA])
		}
	})

	t.Run("nil removes every link", func(t *testing.T) {
		link(t, nil, 0, 2)
		if got := tagSlugs(t, q, post.ID); len(got) != 0 {
			t.Errorf("tags = %v, want none", got)
		}
	})

	if got, want := tagSlugs(t, q, other.ID), []string{"old-a", "old-b"}; !slices.Equal(got, want) {
		t.Errorf("other post's tags = %v, want %v (untouched)", got, want)
	}
}

func tagSlugs(t *testing.T, q *db.Queries, postID uuid.UUID) []string {
	t.Helper()
	rows, err := q.ListTagsForPost(t.Context(), postID)
	if err != nil {
		t.Fatalf("ListTagsForPost: %v", err)
	}
	var out []string
	for _, r := range rows {
		out = append(out, r.Slug)
	}
	return out
}

// version identifies one physical row version: an UPDATE or a delete and
// re-insert writes a new tuple with a new xmin and ctid.
type version struct{ xmin, ctid string }

func linkVersions(t *testing.T, conn db.DBTX, postID uuid.UUID) map[int32]version {
	t.Helper()
	rows, err := conn.Query(t.Context(),
		`SELECT tag_id, xmin::text, ctid::text FROM post_tags WHERE post_id = $1`, postID)
	if err != nil {
		t.Fatalf("read post_tags row versions: %v", err)
	}
	defer rows.Close()
	out := map[int32]version{}
	for rows.Next() {
		var (
			tagID int32
			v     version
		)
		if err := rows.Scan(&tagID, &v.xmin, &v.ctid); err != nil {
			t.Fatalf("scan post_tags row version: %v", err)
		}
		out[tagID] = v
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read post_tags row versions: %v", err)
	}
	return out
}

func TestStoreList(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	dbtest.InsertPost(t, pool, dbtest.WithTags("go", "postgres"))
	dbtest.InsertPost(t, pool, dbtest.Draft(), dbtest.WithTags("drafts-only"))
	dbtest.InsertTag(t, pool, "unused")
	store := tags.NewStore(pool)

	tests := []struct {
		name  string
		scope tags.Scope
		want  []string
	}{
		{"published hides draft-only and unused tags", tags.Published, []string{"go", "postgres"}},
		{"all lists every tag", tags.All, []string{"drafts-only", "go", "postgres", "unused"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := store.List(t.Context(), tc.scope)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			slugs := make([]string, len(got))
			for i, c := range got {
				slugs[i] = c.Slug
			}
			if !slices.Equal(slugs, tc.want) {
				t.Errorf("slugs = %v, want %v", slugs, tc.want)
			}
		})
	}

	t.Run("the zero scope is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := store.List(t.Context(), 0); err == nil {
			t.Error("List(0) = nil error, want one")
		}
	})
}
