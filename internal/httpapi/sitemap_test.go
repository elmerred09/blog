package httpapi

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

func TestGetSitemap(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)
	now := dbtest.Now(t, pool).Truncate(time.Microsecond)

	dbtest.InsertPost(t, pool, dbtest.WithSlug("newer"), dbtest.PublishedAt(now.Add(-time.Hour)))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("older"), dbtest.PublishedAt(now.Add(-2*time.Hour)))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), dbtest.Deleted())

	rec := get(t, h, "/api/v1/sitemap")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}

	raw := decode[map[string]any](t, rec)
	if got, want := keys(raw), []string{"items"}; !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
	page := decode[sitemapJSON](t, rec)
	if len(page.Items) != 2 || page.Items[0].Slug != "newer" || page.Items[1].Slug != "older" {
		t.Fatalf("items = %+v, want newer then older", page.Items)
	}
	first, _ := raw["items"].([]any)
	item, _ := first[0].(map[string]any)
	if got, want := keys(item), []string{"last_modified", "slug"}; !slices.Equal(got, want) {
		t.Errorf("item keys = %v, want %v", got, want)
	}
	if at, _ := item["last_modified"].(string); !strings.HasSuffix(at, "Z") {
		t.Errorf("last_modified = %q, want RFC 3339 in UTC", at)
	}
}

func TestGetSitemapEmpty(t *testing.T) {
	t.Parallel()
	h, _ := postsAPI(t)

	rec := get(t, h, "/api/v1/sitemap")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	page := decode[sitemapJSON](t, rec)
	if page.Items == nil || len(page.Items) != 0 {
		t.Errorf("items = %#v, want a non-nil empty slice", page.Items)
	}
	if !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Errorf("body = %s, want an empty items array", rec.Body)
	}
}
