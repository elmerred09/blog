package querytest

import (
	"testing"
	"time"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

func TestFixtures(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	now := dbtest.Now(t, pool)

	tests := []struct {
		name string
		opts []dbtest.PostOption
		// state is computed by the database, using the same rules as the queries.
		wantState string
	}{
		{"default is published", nil, "published"},
		{"published ago", []dbtest.PostOption{dbtest.PublishedAgo(48 * time.Hour)}, "published"},
		{"published at exact time", []dbtest.PostOption{dbtest.PublishedAt(now.Add(-time.Minute))}, "published"},
		{"draft", []dbtest.PostOption{dbtest.Draft()}, "draft"},
		{"scheduled", []dbtest.PostOption{dbtest.ScheduledIn(time.Hour)}, "scheduled"},
		{"deleted", []dbtest.PostOption{dbtest.Deleted()}, "deleted"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := dbtest.InsertPost(t, pool, tc.opts...)

			var state string
			err := pool.QueryRow(t.Context(), `
SELECT CASE
    WHEN deleted_at IS NOT NULL THEN 'deleted'
    WHEN published_at IS NULL THEN 'draft'
    WHEN published_at > now() THEN 'scheduled'
    ELSE 'published'
END
FROM posts WHERE id = $1`, p.ID).Scan(&state)
			if err != nil {
				t.Fatalf("read state: %v", err)
			}
			if state != tc.wantState {
				t.Errorf("state = %q, want %q", state, tc.wantState)
			}
		})
	}

	t.Run("exact time is kept", func(t *testing.T) {
		at := now.Add(-24 * time.Hour).Truncate(time.Microsecond)
		p := dbtest.InsertPost(t, pool, dbtest.PublishedAt(at))
		if p.PublishedAt == nil || !p.PublishedAt.Equal(at) {
			t.Errorf("published_at = %v, want %v", p.PublishedAt, at)
		}
	})

	t.Run("tags are created once and linked", func(t *testing.T) {
		a := dbtest.InsertPost(t, pool, dbtest.WithTags("go", "postgres"))
		b := dbtest.InsertPost(t, pool, dbtest.WithTags("go"))
		goID := dbtest.InsertTag(t, pool, "go")

		var tagRows, links int
		err := pool.QueryRow(t.Context(), `
SELECT (SELECT count(*) FROM tags WHERE slug = 'go'),
       (SELECT count(*) FROM post_tags WHERE tag_id = $1 AND post_id IN ($2, $3))`,
			goID, a.ID, b.ID).Scan(&tagRows, &links)
		if err != nil {
			t.Fatalf("count tags: %v", err)
		}
		if tagRows != 1 || links != 2 {
			t.Errorf("go tag rows = %d, links = %d; want 1 and 2", tagRows, links)
		}
	})
}
