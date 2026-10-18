package httpapi

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/tags"
)

// postsAPI returns the router over a fresh database, plus the pool for fixtures.
func postsAPI(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool, _ := env.NewDB(t)
	log, _ := testLogger()
	return New(Deps{Logger: log, DB: pool, Posts: post.NewService(pool), Tags: tags.NewStore(pool)}), pool
}

// keys returns a JSON object's keys, sorted.
func keys(m map[string]any) []string {
	return slices.Sorted(maps.Keys(m))
}

func TestListPostsShape(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)

	dbtest.InsertPost(t, pool, dbtest.WithSlug("tagged"), dbtest.WithTitle("Tagged"),
		dbtest.PublishedAgo(time.Hour), dbtest.WithTags("search", "go"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("untagged"), dbtest.PublishedAgo(2*time.Hour))

	rec := get(t, h, "/api/v1/posts")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}

	raw := decode[map[string]any](t, rec)
	if got, want := keys(raw), []string{"items", "next_cursor"}; !slices.Equal(got, want) {
		t.Errorf("page keys = %v, want %v", got, want)
	}
	if raw["next_cursor"] != nil {
		t.Errorf("next_cursor = %v, want null on the only page", raw["next_cursor"])
	}
	items, _ := raw["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %v, want 2", raw["items"])
	}
	first, _ := items[0].(map[string]any)
	if got, want := keys(first), []string{"published_at", "slug", "summary", "tags", "title"}; !slices.Equal(got, want) {
		t.Errorf("item keys = %v, want %v", got, want)
	}
	if untagged, _ := items[1].(map[string]any); untagged["tags"] == nil {
		t.Errorf("untagged post has tags null, want []")
	}

	page := decode[pageJSON](t, rec)
	got := page.Items[0]
	if got.Slug != "tagged" || got.Title != "Tagged" || got.Summary != "Summary of tagged" {
		t.Errorf("first item = %+v", got)
	}
	if want := []tagJSON{{Slug: "go", Name: "go"}, {Slug: "search", Name: "search"}}; !slices.Equal(got.Tags, want) {
		t.Errorf("tags = %v, want %v", got.Tags, want)
	}
	if at, _ := first["published_at"].(string); !strings.HasSuffix(at, "Z") {
		t.Errorf("published_at = %q, want RFC 3339 in UTC", at)
	}
}

func TestListPostsEmpty(t *testing.T) {
	t.Parallel()
	h, _ := postsAPI(t)

	rec := get(t, h, "/api/v1/posts")

	if body := strings.TrimSpace(rec.Body.String()); body != `{"items":[],"next_cursor":null}` {
		t.Errorf("body = %s, want an empty items array and null next_cursor", body)
	}
}

func TestListPostsDefaultLimit(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)
	for i := range post.DefaultLimit + 1 {
		dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Duration(i+1)*time.Minute))
	}

	page := decode[pageJSON](t, get(t, h, "/api/v1/posts"))

	if len(page.Items) != post.DefaultLimit || page.NextCursor == nil {
		t.Errorf("got %d items, next_cursor %v; want %d and a cursor",
			len(page.Items), page.NextCursor, post.DefaultLimit)
	}
}

func TestListPostsFollowsNextCursor(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)

	var published []dbtest.Post
	for i := range 25 {
		published = append(published, dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Duration(i+1)*time.Hour)))
	}
	dbtest.InsertPost(t, pool, dbtest.Draft())

	var (
		slugs  []string
		pages  int
		target = "/api/v1/posts?limit=7"
	)
	for target != "" {
		if pages++; pages > 10 {
			t.Fatalf("still paginating after %d pages", pages-1)
		}
		rec := get(t, h, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d; body %s", target, rec.Code, rec.Body)
		}
		page := decode[pageJSON](t, rec)
		for _, it := range page.Items {
			slugs = append(slugs, it.Slug)
		}
		target = ""
		if page.NextCursor != nil {
			target = "/api/v1/posts?limit=7&cursor=" + url.QueryEscape(*page.NextCursor)
		}
	}

	if want := dbtest.Slugs(dbtest.NewestFirst(published)); !slices.Equal(slugs, want) {
		t.Errorf("slugs across pages:\n got %v\nwant %v", slugs, want)
	}
	if pages != 4 {
		t.Errorf("pages = %d, want 4 (25 posts at 7 per page)", pages)
	}
}

func TestListPostsRejectsBadQuery(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)
	dbtest.InsertPost(t, pool)

	for _, query := range []string{
		"limit=0",
		"limit=-1",
		"limit=51",
		"limit=abc",
		"limit=",
		"limit=1.5",
		"cursor=garbage",
		"cursor=",
	} {
		t.Run(query, func(t *testing.T) {
			wantError(t, get(t, h, "/api/v1/posts?"+query), http.StatusBadRequest, codeInvalidRequest)
		})
	}
}

func TestGetPost(t *testing.T) {
	t.Parallel()
	h, pool := postsAPI(t)

	dbtest.InsertPost(t, pool, dbtest.WithSlug("tagged"), dbtest.WithTitle("Tagged"), dbtest.WithTags("postgres", "go"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("untagged"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), dbtest.Deleted())

	t.Run("published post", func(t *testing.T) {
		rec := get(t, h, "/api/v1/posts/tagged")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
		}

		raw := decode[map[string]any](t, rec)
		want := []string{"body_html", "published_at", "slug", "summary", "tags", "title", "updated_at"}
		if got := keys(raw); !slices.Equal(got, want) {
			t.Errorf("keys = %v, want %v", got, want)
		}

		got := decode[postJSON](t, rec)
		if got.Slug != "tagged" || got.Title != "Tagged" || got.BodyHTML != "<h1>Tagged</h1>" || got.UpdatedAt.IsZero() {
			t.Errorf("post = %+v", got)
		}
		if want := []tagJSON{{Slug: "go", Name: "go"}, {Slug: "postgres", Name: "postgres"}}; !slices.Equal(got.Tags, want) {
			t.Errorf("tags = %v, want %v", got.Tags, want)
		}
	})

	t.Run("untagged post has tags []", func(t *testing.T) {
		raw := decode[map[string]any](t, get(t, h, "/api/v1/posts/untagged"))
		if tags, ok := raw["tags"].([]any); !ok || len(tags) != 0 {
			t.Errorf("tags = %#v, want []", raw["tags"])
		}
	})

	for _, slug := range []string{"draft", "scheduled", "deleted", "missing"} {
		t.Run(slug+" is 404", func(t *testing.T) {
			wantError(t, get(t, h, "/api/v1/posts/"+slug), http.StatusNotFound, codeNotFound)
		})
	}
}

func TestPostsDatabaseErrorIs500(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	pool.Close()
	log, buf := testLogger()
	h := New(Deps{Logger: log, DB: pool, Posts: post.NewService(pool)})

	tests := []struct {
		target  string
		wantLog string
	}{
		{"/api/v1/posts", "list posts: closed pool"},
		{"/api/v1/posts/anything", `get post "anything": closed pool`},
	}
	for _, tc := range tests {
		buf.Reset()
		rec := get(t, h, tc.target)

		wantError(t, rec, http.StatusInternalServerError, codeInternal)
		if line := requestLine(t, buf); line["error"] != tc.wantLog {
			t.Errorf("GET %s: logged error = %q, want %q", tc.target, line["error"], tc.wantLog)
		}
	}
}
