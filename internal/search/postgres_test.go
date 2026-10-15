package search_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/search"
)

func newSearcher(t *testing.T) (*search.Postgres, *pgxpool.Pool) {
	t.Helper()
	pool, _ := env.NewDB(t)
	return search.NewPostgres(pool), pool
}

func run(t *testing.T, s search.Searcher, text string, limit int) []search.Result {
	t.Helper()
	results, err := s.Search(t.Context(), search.Query{Text: text, Limit: limit})
	if err != nil {
		t.Fatalf("Search(%q): %v", text, err)
	}
	if results == nil {
		t.Fatalf("Search(%q) returned nil, want a non-nil slice", text)
	}
	return results
}

func slugs(results []search.Result) []string {
	s := make([]string, len(results))
	for i, r := range results {
		s[i] = r.Slug
	}
	return s
}

// matched returns the text of the snippet's matched fragments.
func matched(r search.Result) []string {
	var m []string
	for _, f := range r.Snippet {
		if f.Match {
			m = append(m, f.Text)
		}
	}
	return m
}

func snippetText(r search.Result) string {
	var b strings.Builder
	for _, f := range r.Snippet {
		b.WriteString(f.Text)
	}
	return b.String()
}

func TestSearchRanking(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)

	// Each case uses its own word, so the cases don't match each other's posts.
	dbtest.InsertPost(t, pool, dbtest.WithSlug("title-hit"), dbtest.WithTitle("Zebra crossings"),
		dbtest.WithBody("Notes on road design."))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("body-hit"), dbtest.WithTitle("Road design"),
		dbtest.WithBody("There is a zebra crossing outside."))

	// One mention each, so only length separates them. The long post is newer,
	// so it would come first if the ranks tied.
	dbtest.InsertPost(t, pool, dbtest.WithSlug("focused"), dbtest.WithTitle("Notes"),
		dbtest.WithBody("A walrus appeared."), dbtest.PublishedAgo(2*time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("diluted"), dbtest.WithTitle("Notes"),
		dbtest.WithBody("A walrus appeared. "+strings.Repeat("Unrelated filler about other topics. ", 200)),
		dbtest.PublishedAgo(time.Hour))

	tests := []struct {
		name string
		q    string
		want []string
	}{
		{"a title match outranks a body-only match", "zebra", []string{"title-hit", "body-hit"}},
		{"a short focused post outranks a long one that mentions the word once", "walrus", []string{"focused", "diluted"}},
		{"stemming matches other word forms", "zebras", []string{"title-hit", "body-hit"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := slugs(run(t, s, tc.q, search.DefaultLimit)); !slices.Equal(got, tc.want) {
				t.Errorf("Search(%q) = %v, want %v", tc.q, got, tc.want)
			}
		})
	}
}

func TestSearchTiesAreNewestFirst(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)

	now := dbtest.Now(t, pool).Truncate(time.Microsecond)
	var posts []dbtest.Post
	insert := func(at time.Time) {
		posts = append(posts, dbtest.InsertPost(t, pool, dbtest.WithTitle("Same"),
			dbtest.WithBody("Identical body about otters."), dbtest.PublishedAt(at)))
	}
	insert(now.Add(-3 * time.Hour))
	insert(now.Add(-1 * time.Hour))
	// Three posts with the same rank and published_at: only the id orders them.
	for range 3 {
		insert(now.Add(-2 * time.Hour))
	}

	want := dbtest.Slugs(dbtest.NewestFirst(posts))
	for range 3 {
		if got := slugs(run(t, s, "otters", search.DefaultLimit)); !slices.Equal(got, want) {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestSearchOnlyPublished(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)

	body := dbtest.WithBody("All about the narwhal.")
	dbtest.InsertPost(t, pool, dbtest.WithSlug("published"), body)
	dbtest.InsertPost(t, pool, dbtest.WithSlug("draft"), body, dbtest.Draft())
	dbtest.InsertPost(t, pool, dbtest.WithSlug("scheduled"), body, dbtest.ScheduledIn(time.Hour))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("deleted"), body, dbtest.Deleted())

	if got := slugs(run(t, s, "narwhal", search.DefaultLimit)); !slices.Equal(got, []string{"published"}) {
		t.Errorf("results = %v, want only the published post", got)
	}
}

func TestSearchLimit(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)
	for range 5 {
		dbtest.InsertPost(t, pool, dbtest.WithBody("Lemurs everywhere."))
	}

	if got := run(t, s, "lemurs", 3); len(got) != 3 {
		t.Errorf("got %d results at limit 3, want 3", len(got))
	}
}

func TestSearchNoResults(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)
	dbtest.InsertPost(t, pool, dbtest.WithTitle("The end"), dbtest.WithBody("And then there were none."))

	for _, q := range []string{
		"aardvark",
		"the and", // only stopwords: an empty tsquery matches nothing
		`"" - or`, // websearch syntax with no words
	} {
		t.Run(q, func(t *testing.T) {
			if got := run(t, s, q, search.DefaultLimit); len(got) != 0 {
				t.Errorf("results = %v, want none", slugs(got))
			}
		})
	}
}

func TestSearchWebsearchSyntax(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)
	dbtest.InsertPost(t, pool, dbtest.WithSlug("keyset"), dbtest.WithBody("Use keyset pagination."))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("both"), dbtest.WithBody("Keyset pagination beats offset pagination."))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("apart"), dbtest.WithBody("Keyset is a word. So is pagination."))

	tests := []struct {
		q    string
		want []string
	}{
		{"keyset -offset", []string{"apart", "keyset"}},
		{`"keyset pagination"`, []string{"both", "keyset"}},
	}
	for _, tc := range tests {
		t.Run(tc.q, func(t *testing.T) {
			got := slugs(run(t, s, tc.q, search.DefaultLimit))
			slices.Sort(got)
			if !slices.Equal(got, tc.want) {
				t.Errorf("Search(%q) = %v, want %v (any order)", tc.q, got, tc.want)
			}
		})
	}
}

func TestSearchSnippet(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)
	dbtest.InsertPost(t, pool, dbtest.WithSlug("body"), dbtest.WithTitle("Notes"),
		dbtest.WithBody("Some text first. Then Pelicans arrive, and more text after."))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("title-only"), dbtest.WithTitle("Pelicans"),
		dbtest.WithBody("Nothing about birds here."))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("html"), dbtest.WithTitle("Markup"),
		dbtest.WithBody("The pelican wins whenever price < cost && margin > zero <script>alert(1)</script> overall."))

	byslug := map[string]search.Result{}
	for _, r := range run(t, s, "pelican", search.DefaultLimit) {
		byslug[r.Slug] = r
	}

	t.Run("body match highlights the word as written", func(t *testing.T) {
		r := byslug["body"]
		if got := matched(r); !slices.Equal(got, []string{"Pelicans"}) {
			t.Errorf("matched fragments = %q, want [Pelicans]; snippet %+v", got, r.Snippet)
		}
		if text := snippetText(r); !strings.Contains(text, "Then Pelicans arrive") {
			t.Errorf("snippet text = %q, want the words around the match", text)
		}
	})

	t.Run("title-only match has an empty snippet", func(t *testing.T) {
		r, ok := byslug["title-only"]
		if !ok {
			t.Fatal("title-only post missing from results")
		}
		if r.Snippet == nil || len(r.Snippet) != 0 {
			t.Errorf("snippet = %#v, want empty and non-nil", r.Snippet)
		}
	})

	// ts_headline's parser treats anything shaped like a tag as markup and
	// leaves it out; other characters come back as written.
	t.Run("tags are dropped, other symbols kept", func(t *testing.T) {
		text := snippetText(byslug["html"])
		if strings.Contains(text, "<script>") {
			t.Errorf("snippet text = %q, want the tag dropped", text)
		}
		if !strings.Contains(text, "price < cost && margin > zero") {
			t.Errorf("snippet text = %q, want lone < & > verbatim", text)
		}
	})
}

func TestSearchResultFields(t *testing.T) {
	t.Parallel()
	s, pool := newSearcher(t)
	p := dbtest.InsertPost(t, pool, dbtest.WithSlug("tagged"), dbtest.WithTitle("Badgers"),
		dbtest.WithSummary("All about badgers."), dbtest.WithBody("Badgers dig."), dbtest.WithTags("zoo", "animals"))
	dbtest.InsertPost(t, pool, dbtest.WithSlug("untagged"), dbtest.WithTitle("More badgers"), dbtest.WithBody("Badgers sleep."))

	results := run(t, s, "badgers", search.DefaultLimit)
	byslug := map[string]search.Result{}
	for _, r := range results {
		byslug[r.Slug] = r
	}

	got := byslug["tagged"]
	if got.Title != "Badgers" || got.Summary != "All about badgers." || !got.PublishedAt.Equal(*p.PublishedAt) {
		t.Errorf("result = %+v", got.ListItem)
	}
	if want := []post.Tag{{Slug: "animals", Name: "animals"}, {Slug: "zoo", Name: "zoo"}}; !slices.Equal(got.Tags, want) {
		t.Errorf("tags = %v, want %v (sorted by slug)", got.Tags, want)
	}
	if tags := byslug["untagged"].Tags; tags == nil || len(tags) != 0 {
		t.Errorf("untagged tags = %#v, want empty and non-nil", tags)
	}
}

func TestSearchInvalidQuery(t *testing.T) {
	t.Parallel()
	// A closed pool fails any query, so this also shows validation runs first.
	pool, _ := env.NewDB(t)
	pool.Close()
	s := search.NewPostgres(pool)

	_, err := s.Search(t.Context(), search.Query{Text: "  ", Limit: 10})
	var invalid *search.ValidationError
	if !errors.As(err, &invalid) {
		t.Errorf("err = %v, want a *search.ValidationError", err)
	}
}

func TestSearchDatabaseError(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	pool.Close()

	_, err := search.NewPostgres(pool).Search(t.Context(), search.Query{Text: "go", Limit: 10})
	var invalid *search.ValidationError
	if err == nil || errors.As(err, &invalid) {
		t.Errorf("err = %v, want a database error", err)
	}
}
