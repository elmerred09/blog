package admin_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/render"
)

func TestRerender(t *testing.T) {
	t.Parallel()
	pool, svc := newService(t)

	// dbtest stores <h1>Title</h1>, which is not what the renderer makes of
	// "# Title", so every live post starts out stale. More than one batch
	// checks that the walk continues past the first.
	posts := make([]dbtest.Post, admin.RerenderBatch+1)
	for i := range posts {
		posts[i] = dbtest.InsertPost(t, pool, dbtest.WithBody("## Section"))
	}
	draft := dbtest.InsertPost(t, pool, dbtest.Draft(), dbtest.WithBody("## Section"))
	deleted := dbtest.InsertPost(t, pool, dbtest.Deleted(), dbtest.WithBody("## Section"))
	before := updatedAt(t, pool, posts[0].ID)

	got, err := svc.Rerender(t.Context())
	if err != nil {
		t.Fatalf("Rerender: %v", err)
	}
	live := len(posts) + 1
	if got.Checked != live || got.Updated != live {
		t.Errorf("first run = %+v, want %d checked and updated", got, live)
	}

	want, err := render.New().Render([]byte("## Section"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, id := range []uuid.UUID{posts[0].ID, posts[len(posts)-1].ID, draft.ID} {
		if html := bodyHTML(t, pool, id); html != want {
			t.Errorf("post %s body_html = %q, want %q", id, html, want)
		}
	}
	if html := bodyHTML(t, pool, deleted.ID); html == want {
		t.Error("deleted post was rerendered")
	}
	if after := updatedAt(t, pool, posts[0].ID); !after.Equal(before) {
		t.Errorf("updated_at moved from %v to %v; it is the editor's lock token", before, after)
	}

	again, err := svc.Rerender(t.Context())
	if err != nil {
		t.Fatalf("second Rerender: %v", err)
	}
	if again.Checked != live || again.Updated != 0 {
		t.Errorf("second run = %+v, want %d checked and none updated", again, live)
	}
}

func bodyHTML(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var html string
	if err := pool.QueryRow(t.Context(), "SELECT body_html FROM posts WHERE id = $1", id).Scan(&html); err != nil {
		t.Fatalf("read body_html: %v", err)
	}
	return html
}

func updatedAt(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) time.Time {
	t.Helper()
	var at time.Time
	if err := pool.QueryRow(t.Context(), "SELECT updated_at FROM posts WHERE id = $1", id).Scan(&at); err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	return at
}
