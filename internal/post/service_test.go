package post_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/post"
)

// listAll follows Next from the first page to the last and returns every item
// in the order the pages returned them, plus the number of pages.
func listAll(t *testing.T, svc *post.Service, limit int) ([]post.ListItem, int) {
	t.Helper()

	const maxPages = 100 // guard against a cursor that never advances
	var (
		all   []post.ListItem
		after *post.Cursor
	)
	for pages := 1; pages <= maxPages; pages++ {
		page, err := svc.List(t.Context(), post.ListParams{After: after, Limit: limit})
		if err != nil {
			t.Fatalf("List page %d: %v", pages, err)
		}
		all = append(all, page.Items...)
		if page.Next == nil {
			return all, pages
		}
		if len(page.Items) != limit {
			t.Fatalf("page %d has %d items and a Next; only full pages should have one", pages, len(page.Items))
		}
		after = page.Next
	}
	t.Fatalf("still paginating after %d pages", maxPages)
	return nil, 0
}

// newestFirst returns the posts' slugs in list order: published_at descending,
// then id descending (Postgres compares uuids bytewise).
func newestFirst(posts []dbtest.Post) []string {
	sorted := slices.Clone(posts)
	slices.SortFunc(sorted, func(a, b dbtest.Post) int {
		if c := b.PublishedAt.Compare(*a.PublishedAt); c != 0 {
			return c
		}
		return bytes.Compare(b.ID[:], a.ID[:])
	})
	slugs := make([]string, len(sorted))
	for i, p := range sorted {
		slugs[i] = p.Slug
	}
	return slugs
}

func itemSlugs(items []post.ListItem) []string {
	slugs := make([]string, len(items))
	for i, it := range items {
		slugs[i] = it.Slug
	}
	return slugs
}

func TestListWalksEveryPublishedPost(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)

	now := dbtest.Now(t, pool).Truncate(time.Microsecond)
	var published []dbtest.Post
	insert := func(at time.Time) {
		published = append(published, dbtest.InsertPost(t, pool, dbtest.PublishedAt(at)))
	}
	// 23 posts at limit 10: positions 9-13 share a published_at, so the tie
	// crosses the boundary between pages 1 and 2 and only the id breaks it.
	for i := 1; i <= 8; i++ {
		insert(now.Add(-time.Duration(i) * time.Hour))
	}
	for range 5 {
		insert(now.Add(-9 * time.Hour))
	}
	for i := 14; i <= 23; i++ {
		insert(now.Add(-time.Duration(i) * time.Hour))
	}

	dbtest.InsertPost(t, pool, dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.Deleted())

	got, pages := listAll(t, svc, 10)
	if want := newestFirst(published); !slices.Equal(itemSlugs(got), want) {
		t.Errorf("items across pages:\n got %v\nwant %v", itemSlugs(got), want)
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3 (23 posts at 10 per page)", pages)
	}
}

func TestListPageBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("no posts gives an empty, non-nil page", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)

		page, err := post.NewService(pool).List(t.Context(), post.ListParams{Limit: 10})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if page.Items == nil || len(page.Items) != 0 || page.Next != nil {
			t.Errorf("page = %+v, want non-nil empty Items and no Next", page)
		}
	})

	t.Run("exactly limit posts is one page without Next", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		for i := range 10 {
			dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Duration(i+1)*time.Minute))
		}

		got, pages := listAll(t, post.NewService(pool), 10)
		if len(got) != 10 || pages != 1 {
			t.Errorf("got %d items over %d pages, want 10 over 1", len(got), pages)
		}
	})
}

func TestListItemFields(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)

	tagged := dbtest.InsertPost(t, pool,
		dbtest.WithSlug("tagged"), dbtest.WithTitle("Tagged"),
		dbtest.PublishedAgo(time.Hour), dbtest.WithTags("search", "go"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("untagged"), dbtest.PublishedAgo(2*time.Hour))

	page, err := svc.List(t.Context(), post.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := itemSlugs(page.Items); !slices.Equal(got, []string{"tagged", "untagged"}) {
		t.Fatalf("items = %v, want [tagged untagged]", got)
	}

	first := page.Items[0]
	if first.Title != "Tagged" || first.Summary != "Summary of tagged" || !first.PublishedAt.Equal(*tagged.PublishedAt) {
		t.Errorf("item = %+v, want title Tagged, summary %q, published_at %v",
			first, "Summary of tagged", *tagged.PublishedAt)
	}
	if want := []post.Tag{{Slug: "go", Name: "go"}, {Slug: "search", Name: "search"}}; !slices.Equal(first.Tags, want) {
		t.Errorf("tags = %v, want %v (sorted by slug)", first.Tags, want)
	}
	if tags := page.Items[1].Tags; tags == nil || len(tags) != 0 {
		t.Errorf("untagged post tags = %#v, want a non-nil empty slice", tags)
	}
}

func TestListLimit(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)
	dbtest.InsertPost(t, pool)

	tests := []struct {
		limit   int
		wantErr error
	}{
		{-1, post.ErrInvalidLimit},
		{0, post.ErrInvalidLimit},
		{1, nil},
		{post.MaxLimit, nil},
		{post.MaxLimit + 1, post.ErrInvalidLimit},
	}
	for _, tc := range tests {
		_, err := svc.List(t.Context(), post.ListParams{Limit: tc.limit})
		if !errors.Is(err, tc.wantErr) {
			t.Errorf("List(limit %d) error = %v, want %v", tc.limit, err, tc.wantErr)
		}
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)

	tagged := dbtest.InsertPost(t, pool,
		dbtest.WithSlug("tagged"), dbtest.WithTitle("Tagged"), dbtest.WithTags("postgres", "go"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("untagged"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), dbtest.Deleted())

	t.Run("published post with tags", func(t *testing.T) {
		got, err := svc.Get(t.Context(), "tagged")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Slug != "tagged" || got.Title != "Tagged" || got.Summary != "Summary of tagged" ||
			got.BodyHTML != "<h1>Tagged</h1>" || !got.PublishedAt.Equal(*tagged.PublishedAt) || got.UpdatedAt.IsZero() {
			t.Errorf("post = %+v", got)
		}
		if want := []post.Tag{{Slug: "go", Name: "go"}, {Slug: "postgres", Name: "postgres"}}; !slices.Equal(got.Tags, want) {
			t.Errorf("tags = %v, want %v", got.Tags, want)
		}
	})

	t.Run("untagged post has a non-nil empty tag slice", func(t *testing.T) {
		got, err := svc.Get(t.Context(), "untagged")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Tags == nil || len(got.Tags) != 0 {
			t.Errorf("tags = %#v, want a non-nil empty slice", got.Tags)
		}
	})

	for _, slug := range []string{"draft", "scheduled", "deleted", "missing"} {
		t.Run(slug+" is not found", func(t *testing.T) {
			if _, err := svc.Get(t.Context(), slug); !errors.Is(err, post.ErrNotFound) {
				t.Errorf("Get(%q) error = %v, want ErrNotFound", slug, err)
			}
		})
	}
}
