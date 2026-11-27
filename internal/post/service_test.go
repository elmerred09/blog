package post_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

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
		page, err := svc.List(t.Context(), post.ListParams{Cursor: after, Limit: limit})
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
	if want := dbtest.Slugs(dbtest.NewestFirst(published)); !slices.Equal(itemSlugs(got), want) {
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
		if page.Items == nil || len(page.Items) != 0 || page.Next != nil || page.Prev != nil {
			t.Errorf("page = %+v, want non-nil empty Items and no cursors", page)
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
		dbtest.PublishedAgo(time.Hour), dbtest.WithTags("search", "go"),
		dbtest.WithBody(strings.Repeat("word ", 231)))
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
	if got := first.ReadingMinutes; got != 2 {
		t.Errorf("reading minutes for 231 words = %d, want 2", got)
	}
	if got := page.Items[1].ReadingMinutes; got != 1 {
		t.Errorf("reading minutes for the default two-word body = %d, want 1", got)
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
		dbtest.WithSlug("tagged"), dbtest.WithTitle("Tagged"), dbtest.WithTags("postgres", "go"),
		dbtest.WithBody(strings.Repeat("word\n", 461)))
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
		if got.ReadingMinutes != 3 {
			t.Errorf("reading minutes for 461 words = %d, want 3", got.ReadingMinutes)
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

func TestAdjacent(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)

	dbtest.InsertPost(t, pool, dbtest.WithSlug("oldest"), dbtest.WithTitle("Oldest"), dbtest.PublishedAgo(3*time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("middle"), dbtest.WithTitle("Middle"), dbtest.PublishedAgo(2*time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), dbtest.PublishedAgo(90*time.Minute), dbtest.Deleted())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("newest"), dbtest.WithTitle("Newest"), dbtest.PublishedAgo(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft())

	oldest := &post.Neighbor{Slug: "oldest", Title: "Oldest"}
	middle := &post.Neighbor{Slug: "middle", Title: "Middle"}
	newest := &post.Neighbor{Slug: "newest", Title: "Newest"}

	tests := []struct {
		slug         string
		newer, older *post.Neighbor
	}{
		{"oldest", middle, nil},
		{"middle", newest, oldest},
		{"newest", nil, middle},
	}
	for _, tc := range tests {
		t.Run(tc.slug, func(t *testing.T) {
			got, err := svc.Adjacent(t.Context(), tc.slug)
			if err != nil {
				t.Fatalf("Adjacent: %v", err)
			}
			if !equalNeighbor(got.Newer, tc.newer) || !equalNeighbor(got.Older, tc.older) {
				t.Errorf("Adjacent(%q) = newer %+v, older %+v; want newer %+v, older %+v",
					tc.slug, got.Newer, got.Older, tc.newer, tc.older)
			}
		})
	}

	for _, slug := range []string{"draft", "scheduled", "deleted", "missing"} {
		t.Run(slug+" is not found", func(t *testing.T) {
			if _, err := svc.Adjacent(t.Context(), slug); !errors.Is(err, post.ErrNotFound) {
				t.Errorf("Adjacent(%q) error = %v, want ErrNotFound", slug, err)
			}
		})
	}
}

// Posts published at the same moment are ordered by id, as in the list, so
// each still has the other as a neighbor.
func TestAdjacentTies(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)

	at := dbtest.Now(t, pool).Add(-time.Hour)
	first := dbtest.InsertPost(t, pool, dbtest.PublishedAt(at))
	second := dbtest.InsertPost(t, pool, dbtest.PublishedAt(at))
	if first.ID.String() > second.ID.String() {
		first, second = second, first
	}

	got, err := svc.Adjacent(t.Context(), first.Slug)
	if err != nil {
		t.Fatalf("Adjacent: %v", err)
	}
	if got.Older != nil || got.Newer == nil || got.Newer.Slug != second.Slug {
		t.Errorf("Adjacent(lower id) = newer %+v, older %+v; want newer %q", got.Newer, got.Older, second.Slug)
	}

	got, err = svc.Adjacent(t.Context(), second.Slug)
	if err != nil {
		t.Fatalf("Adjacent: %v", err)
	}
	if got.Newer != nil || got.Older == nil || got.Older.Slug != first.Slug {
		t.Errorf("Adjacent(higher id) = newer %+v, older %+v; want older %q", got.Newer, got.Older, first.Slug)
	}
}

func equalNeighbor(a, b *post.Neighbor) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// walk lists pages from start, following Next (dir After) or Prev (dir Before)
// until that direction runs out.
func walk(t *testing.T, svc *post.Service, start *post.Cursor, dir post.Direction, limit int) []post.Page {
	t.Helper()

	cursor := start
	var pages []post.Page
	for range 100 {
		page, err := svc.List(t.Context(), post.ListParams{Cursor: cursor, Limit: limit})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		pages = append(pages, page)
		if dir == post.Before {
			cursor = page.Prev
		} else {
			cursor = page.Next
		}
		if cursor == nil {
			return pages
		}
	}
	t.Fatal("still paginating after 100 pages")
	return nil
}

func sameSlugPages(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !slices.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func pageSlugs(pages []post.Page) [][]string {
	out := make([][]string, len(pages))
	for i, page := range pages {
		out[i] = itemSlugs(page.Items)
	}
	return out
}

// publishSeries inserts n published posts. Five of them share a published_at,
// and at limit 10 that tie crosses the boundary between the first two pages.
func publishSeries(t *testing.T, pool *pgxpool.Pool, n int) {
	t.Helper()

	now := dbtest.Now(t, pool).Truncate(time.Microsecond)
	for i := 1; i <= n; i++ {
		at := now.Add(-time.Duration(i) * time.Hour)
		if i >= 9 && i <= 13 {
			at = now.Add(-9 * time.Hour)
		}
		dbtest.InsertPost(t, pool, dbtest.PublishedAt(at))
	}
}

func TestListBackward(t *testing.T) {
	t.Parallel()

	const limit = 10

	t.Run("forward and backward visit the same pages", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		publishSeries(t, pool, 23)
		svc := post.NewService(pool)

		forward := walk(t, svc, nil, post.After, limit)
		if len(forward) != 3 {
			t.Fatalf("forward pages = %d, want 3", len(forward))
		}
		if forward[0].Prev != nil {
			t.Errorf("first page Prev = %v, want nil", forward[0].Prev)
		}
		if forward[1].Prev == nil || forward[2].Prev == nil {
			t.Error("pages after the first have no Prev")
		}

		back := walk(t, svc, forward[len(forward)-1].Prev, post.Before, limit)
		want := slices.Clone(forward[:len(forward)-1])
		slices.Reverse(want)
		if got, wantSlugs := pageSlugs(back), pageSlugs(want); !sameSlugPages(got, wantSlugs) {
			t.Errorf("backward pages:\n got %v\nwant %v", got, wantSlugs)
		}
		if back[len(back)-1].Prev != nil {
			t.Errorf("walking back to the top left Prev set: %v", back[len(back)-1].Prev)
		}
	})

	t.Run("a backward fetch at the top is the first page", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		publishSeries(t, pool, 23)
		svc := post.NewService(pool)

		forward := walk(t, svc, nil, post.After, limit)
		// The second page's Prev points at its first post. Exactly one page
		// is newer than that post, so the fetch has no extra row.
		page, err := svc.List(t.Context(), post.ListParams{Cursor: forward[1].Prev, Limit: limit})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if page.Prev != nil {
			t.Errorf("Prev = %v, want nil", page.Prev)
		}
		if got, want := itemSlugs(page.Items), itemSlugs(forward[0].Items); !slices.Equal(got, want) {
			t.Errorf("top page = %v, want the first page %v", got, want)
		}
	})

	t.Run("a post published mid-walk shows up on the way back", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		publishSeries(t, pool, 23)
		svc := post.NewService(pool)

		forward := walk(t, svc, nil, post.After, limit)
		added := dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Second))

		back := walk(t, svc, forward[len(forward)-1].Prev, post.Before, limit)
		// The walk ends on the first page, which is where the new post shows.
		// That page can repeat posts from the page before it: the new post
		// made the previous page full, so the one newer than it is a short
		// page and gets replaced by the first page.
		final := itemSlugs(back[len(back)-1].Items)
		if back[len(back)-1].Prev != nil || slices.Index(final, added.Slug) != 0 {
			t.Errorf("final page (prev %v) = %v, want %s first and no Prev", back[len(back)-1].Prev, final, added.Slug)
		}
		seen := map[string]int{}
		for _, page := range back[:len(back)-1] {
			for _, slug := range itemSlugs(page.Items) {
				seen[slug]++
			}
		}
		for _, slug := range itemSlugs(forward[len(forward)-1].Items) {
			seen[slug]++
		}
		if seen[added.Slug] != 0 {
			t.Errorf("new post appears before the final page")
		}
		for slug, n := range seen {
			if n != 1 {
				t.Errorf("%s appears %d times before the final page, want 1", slug, n)
			}
		}
	})
}

func TestSitemapEntries(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	now := dbtest.Now(t, pool).Truncate(time.Microsecond)

	// Edited after it was published: last_modified is the edit.
	editedAt := now.Add(-time.Hour)
	edited := dbtest.InsertPost(t, pool, dbtest.WithSlug("edited"), dbtest.PublishedAt(now.Add(-2*time.Hour)))
	setPostTimes(t, pool, edited.ID, now.Add(-2*time.Hour), editedAt)

	// Edited while scheduled, then published: last_modified is published_at,
	// the moment a reader could first see it.
	publishedAt := now.Add(-30 * time.Minute)
	wentLive := dbtest.InsertPost(t, pool, dbtest.WithSlug("went-live"), dbtest.PublishedAt(publishedAt))
	setPostTimes(t, pool, wentLive.ID, publishedAt, now.Add(-3*time.Hour))

	olderAt := now.Add(-5 * time.Hour)
	older := dbtest.InsertPost(t, pool, dbtest.WithSlug("older"), dbtest.PublishedAt(olderAt))
	setPostTimes(t, pool, older.ID, olderAt, olderAt)

	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), dbtest.PublishedAt(now.Add(-time.Minute)), dbtest.Deleted())

	got, err := post.NewService(pool).SitemapEntries(t.Context())
	if err != nil {
		t.Fatalf("SitemapEntries: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 published posts; got %+v", len(got), got)
	}
	if got[0].Slug != "went-live" || !got[0].LastModified.Equal(publishedAt) {
		t.Errorf("first = %+v, want went-live at %v", got[0], publishedAt)
	}
	if got[1].Slug != "edited" || !got[1].LastModified.Equal(editedAt) {
		t.Errorf("second = %+v, want edited at %v", got[1], editedAt)
	}
	if got[2].Slug != "older" || !got[2].LastModified.Equal(olderAt) {
		t.Errorf("third = %+v, want older at %v", got[2], olderAt)
	}
}

func TestSitemapEntriesEmpty(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)

	got, err := post.NewService(pool).SitemapEntries(t.Context())
	if err != nil {
		t.Fatalf("SitemapEntries: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("entries = %#v, want a non-nil empty slice", got)
	}
}

func setPostTimes(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, published, updated time.Time) {
	t.Helper()
	_, err := pool.Exec(t.Context(),
		`UPDATE posts SET published_at = $2, updated_at = $3 WHERE id = $1`,
		id, published, updated)
	if err != nil {
		t.Fatalf("set times: %v", err)
	}
}

// walkTag lists a tag's pages from start, following Next (dir After) or Prev
// (dir Before) until that direction runs out.
func walkTag(t *testing.T, svc *post.Service, tag string, start *post.Cursor, dir post.Direction, limit int) []post.Page {
	t.Helper()

	cursor := start
	var pages []post.Page
	for range 100 {
		tp, err := svc.ListByTag(t.Context(), tag, post.ListParams{Cursor: cursor, Limit: limit})
		if err != nil {
			t.Fatalf("ListByTag(%q): %v", tag, err)
		}
		if tp.Tag.Slug != tag {
			t.Fatalf("ListByTag(%q) tag = %+v", tag, tp.Tag)
		}
		pages = append(pages, tp.Page)
		if dir == post.Before {
			cursor = tp.Prev
		} else {
			cursor = tp.Next
		}
		if cursor == nil {
			return pages
		}
	}
	t.Fatal("still paginating after 100 pages")
	return nil
}

func TestListByTag(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	svc := post.NewService(pool)

	const limit = 10
	now := dbtest.Now(t, pool).Truncate(time.Microsecond)
	var tagged []dbtest.Post
	for i := 1; i <= 23; i++ {
		at := now.Add(-time.Duration(i) * time.Hour)
		if i >= 9 && i <= 13 {
			at = now.Add(-9 * time.Hour) // a tie across the page 1/2 boundary
		}
		opts := []dbtest.PostOption{dbtest.PublishedAt(at), dbtest.WithTags("go")}
		if i == 1 {
			opts = append(opts, dbtest.WithTags("postgres"))
		}
		tagged = append(tagged, dbtest.InsertPost(t, pool, opts...))
	}
	// Newer than every go post, but tagged otherwise or not listable.
	dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Minute), dbtest.WithTags("other"))
	dbtest.InsertPost(t, pool, dbtest.PublishedAgo(time.Minute))
	dbtest.InsertPost(t, pool, dbtest.Draft(), dbtest.WithTags("go", "secret"))
	dbtest.InsertPost(t, pool, dbtest.ScheduledIn(time.Hour), dbtest.WithTags("go", "secret"))
	dbtest.InsertPost(t, pool, dbtest.Deleted(), dbtest.WithTags("go", "secret"))
	dbtest.InsertTag(t, pool, "unused")

	t.Run("pages forward through exactly the tag's published posts", func(t *testing.T) {
		pages := walkTag(t, svc, "go", nil, post.After, limit)
		var got []string
		for _, p := range pages {
			got = append(got, itemSlugs(p.Items)...)
		}
		if want := dbtest.Slugs(dbtest.NewestFirst(tagged)); !slices.Equal(got, want) {
			t.Errorf("slugs across pages:\n got %v\nwant %v", got, want)
		}
		if len(pages) != 3 {
			t.Errorf("pages = %d, want 3", len(pages))
		}
		if pages[0].Prev != nil {
			t.Errorf("first page Prev = %v, want nil", pages[0].Prev)
		}
		first := pages[0].Items[0]
		if first.Slug != tagged[0].Slug || len(first.Tags) != 2 {
			t.Errorf("first item = %s with tags %v, want %s with go and postgres", first.Slug, first.Tags, tagged[0].Slug)
		}
	})

	t.Run("backward visits the same pages", func(t *testing.T) {
		forward := walkTag(t, svc, "go", nil, post.After, limit)
		back := walkTag(t, svc, "go", forward[len(forward)-1].Prev, post.Before, limit)
		want := slices.Clone(forward[:len(forward)-1])
		slices.Reverse(want)
		if got, wantSlugs := pageSlugs(back), pageSlugs(want); !sameSlugPages(got, wantSlugs) {
			t.Errorf("backward pages:\n got %v\nwant %v", got, wantSlugs)
		}
	})

	t.Run("a backward fetch at the top is the first page", func(t *testing.T) {
		forward := walkTag(t, svc, "go", nil, post.After, limit)
		tp, err := svc.ListByTag(t.Context(), "go", post.ListParams{Cursor: forward[1].Prev, Limit: limit})
		if err != nil {
			t.Fatalf("ListByTag: %v", err)
		}
		if tp.Prev != nil {
			t.Errorf("Prev = %v, want nil", tp.Prev)
		}
		if got, want := itemSlugs(tp.Items), itemSlugs(forward[0].Items); !slices.Equal(got, want) {
			t.Errorf("top page = %v, want the first page %v", got, want)
		}
	})

	oldest := dbtest.NewestFirst(tagged)[len(tagged)-1]
	pastEnd := &post.Cursor{Direction: post.After, PublishedAt: *oldest.PublishedAt, ID: oldest.ID}
	notFound := []struct {
		name   string
		tag    string
		cursor *post.Cursor
	}{
		{"unknown tag", "no-such-tag", nil},
		{"tag without posts", "unused", nil},
		{"tag only on drafts, scheduled and deleted posts", "secret", nil},
		{"cursor past the end", "go", pastEnd},
	}
	for _, tc := range notFound {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ListByTag(t.Context(), tc.tag, post.ListParams{Cursor: tc.cursor, Limit: limit})
			if !errors.Is(err, post.ErrNotFound) {
				t.Errorf("ListByTag(%q) error = %v, want ErrNotFound", tc.tag, err)
			}
		})
	}

	for _, n := range []int{0, post.MaxLimit + 1} {
		t.Run(fmt.Sprintf("limit %d", n), func(t *testing.T) {
			_, err := svc.ListByTag(t.Context(), "go", post.ListParams{Limit: n})
			if !errors.Is(err, post.ErrInvalidLimit) {
				t.Errorf("error = %v, want ErrInvalidLimit", err)
			}
		})
	}
}
