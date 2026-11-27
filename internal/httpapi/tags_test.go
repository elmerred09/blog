package httpapi

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

func TestListPostsByTag(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)

	dbtest.InsertPost(t, pool, dbtest.WithSlug("newest"), dbtest.PublishedAgo(time.Hour), dbtest.WithTags("go", "postgres"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("middle"), dbtest.PublishedAgo(2*time.Hour), dbtest.WithTags("go"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("oldest"), dbtest.PublishedAgo(3*time.Hour), dbtest.WithTags("go"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("other"), dbtest.PublishedAgo(time.Minute), dbtest.WithTags("rust"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft(), dbtest.WithTags("go", "secret"))

	t.Run("first page", func(t *testing.T) {
		rec := get(t, h, "/api/v1/tags/go/posts?limit=2")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
		}
		raw := decode[map[string]any](t, rec)
		if got, want := keys(raw), []string{"items", "next_cursor", "prev_cursor", "tag"}; !slices.Equal(got, want) {
			t.Errorf("keys = %v, want %v", got, want)
		}
		page := decode[tagPageJSON](t, rec)
		if page.Tag != (tagJSON{Slug: "go", Name: "go"}) {
			t.Errorf("tag = %+v", page.Tag)
		}
		if got := slugsOf(page.Items); !slices.Equal(got, []string{"newest", "middle"}) {
			t.Errorf("items = %v, want newest, middle", got)
		}
		if want := []tagJSON{{Slug: "go", Name: "go"}, {Slug: "postgres", Name: "postgres"}}; !slices.Equal(page.Items[0].Tags, want) {
			t.Errorf("newest tags = %v, want %v", page.Items[0].Tags, want)
		}
		if page.PrevCursor != nil || page.NextCursor == nil {
			t.Fatalf("cursors prev %v next %v, want only next", page.PrevCursor, page.NextCursor)
		}

		next := decode[tagPageJSON](t, get(t, h, "/api/v1/tags/go/posts?limit=2&cursor="+*page.NextCursor))
		if got := slugsOf(next.Items); !slices.Equal(got, []string{"oldest"}) {
			t.Errorf("second page = %v, want oldest", got)
		}
		if next.PrevCursor == nil || next.NextCursor != nil {
			t.Errorf("second page cursors prev %v next %v, want only prev", next.PrevCursor, next.NextCursor)
		}
	})

	for _, target := range []string{
		"/api/v1/tags/go/posts?limit=0",
		"/api/v1/tags/go/posts?limit=x",
		"/api/v1/tags/go/posts?cursor=not-a-cursor",
	} {
		t.Run(target+" is 400", func(t *testing.T) {
			wantError(t, get(t, h, target), http.StatusBadRequest, codeInvalidRequest)
		})
	}

	for _, tag := range []string{"missing", "secret"} {
		t.Run(tag+" is 404", func(t *testing.T) {
			wantError(t, get(t, h, "/api/v1/tags/"+tag+"/posts"), http.StatusNotFound, codeNotFound)
		})
	}
}

func slugsOf(items []listItemJSON) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Slug
	}
	return out
}
