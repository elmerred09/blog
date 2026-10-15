package httpapi

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/search"
)

func TestSearchShape(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	log, _ := testLogger()
	h := New(Deps{Logger: log, DB: pool, Search: search.NewPostgres(pool)})

	dbtest.InsertPost(t, pool, dbtest.WithSlug("body-hit"), dbtest.WithTitle("Notes"),
		dbtest.WithBody("Then the pelican arrived."), dbtest.WithTags("birds"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("title-hit"), dbtest.WithTitle("Pelican"),
		dbtest.WithBody("Nothing else here."))

	rec := get(t, h, "/api/v1/search?q=pelican")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}

	raw := decode[map[string][]map[string]any](t, rec)
	if len(raw["items"]) != 2 {
		t.Fatalf("items = %v, want 2", raw["items"])
	}
	want := []string{"published_at", "slug", "snippet", "summary", "tags", "title"}
	if got := keys(raw["items"][0]); !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}

	got := decode[searchResultsJSON](t, rec)
	bySlug := map[string]searchResultJSON{}
	for _, it := range got.Items {
		bySlug[it.Slug] = it
	}
	body := bySlug["body-hit"]
	if !slices.Contains(body.Snippet, fragmentJSON{Text: "pelican", Match: true}) {
		t.Errorf("body-hit snippet = %+v, want a matched pelican fragment", body.Snippet)
	}
	if want := []tagJSON{{Slug: "birds", Name: "birds"}}; !slices.Equal(body.Tags, want) {
		t.Errorf("tags = %v, want %v", body.Tags, want)
	}
	if s := bySlug["title-hit"].Snippet; s == nil || len(s) != 0 {
		t.Errorf("title-hit snippet = %#v, want []", s)
	}

	t.Run("no match is an empty list", func(t *testing.T) {
		raw := decode[map[string]any](t, get(t, h, "/api/v1/search?q=aardvark"))
		if items, ok := raw["items"].([]any); !ok || len(items) != 0 {
			t.Errorf("items = %#v, want []", raw["items"])
		}
	})

	t.Run("limit", func(t *testing.T) {
		got := decode[searchResultsJSON](t, get(t, h, "/api/v1/search?q=pelican&limit=1"))
		if len(got.Items) != 1 {
			t.Errorf("got %d items at limit=1, want 1", len(got.Items))
		}
	})
}

func TestSearchDatabaseErrorIs500(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	pool.Close()
	log, buf := testLogger()
	h := New(Deps{Logger: log, DB: pool, Search: search.NewPostgres(pool)})

	wantError(t, get(t, h, "/api/v1/search?q=go"), http.StatusInternalServerError, codeInternal)
	if line := requestLine(t, buf); line["error"] != "search posts: closed pool" {
		t.Errorf("logged error = %q, want %q", line["error"], "search posts: closed pool")
	}
}

func TestSearchRejectsBadQuery(t *testing.T) {
	t.Parallel()
	log, _ := testLogger()
	// Validation happens before the searcher is called, so none is needed.
	h := New(Deps{Logger: log})

	tooLong := strings.Repeat("a", search.MaxQueryLen+1)
	for _, query := range []string{
		"",
		"q=",
		"q=%20%09",
		"q=" + tooLong,
		"limit=5",
		"q=go&limit=0",
		"q=go&limit=-1",
		"q=go&limit=51",
		"q=go&limit=abc",
		"q=go&limit=",
		"q=go&limit=1.5",
	} {
		t.Run(query, func(t *testing.T) {
			target := "/api/v1/search"
			if query != "" {
				target += "?" + query
			}
			wantError(t, get(t, h, target), http.StatusBadRequest, codeInvalidRequest)
		})
	}

	t.Run("message names the problem", func(t *testing.T) {
		rec := get(t, h, "/api/v1/search?q="+url.QueryEscape(tooLong))
		if msg := decode[errorBody](t, rec).Error.Message; !strings.Contains(msg, "200") {
			t.Errorf("message = %q, want it to give the limit", msg)
		}
	})
}
