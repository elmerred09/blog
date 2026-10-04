package querytest

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

type listed struct {
	ID          uuid.UUID
	PublishedAt *time.Time
}

// pageFunc fetches one page after the given cursor (nil, nil for page 1).
type pageFunc func(afterPublishedAt *time.Time, afterID *uuid.UUID) []listed

// collectPages follows the cursor from page 1 until a short page, and returns
// every row in the order the pages returned them plus the number of pages.
func collectPages(t *testing.T, pageSize int, fetch pageFunc) ([]listed, int) {
	t.Helper()

	var (
		all      []listed
		afterAt  *time.Time
		afterID  *uuid.UUID
		maxPages = 1000 // guard against a cursor that never advances
	)
	for pages := 1; pages <= maxPages; pages++ {
		page := fetch(afterAt, afterID)
		if len(page) > pageSize {
			t.Fatalf("page %d has %d rows, page size is %d", pages, len(page), pageSize)
		}
		all = append(all, page...)
		if len(page) < pageSize {
			return all, pages
		}
		last := page[len(page)-1]
		afterAt, afterID = last.PublishedAt, &last.ID
	}
	t.Fatalf("still paginating after %d pages", maxPages)
	return nil, 0
}

// newestFirst returns posts in the order the list queries promise:
// published_at descending, then id descending. Postgres compares uuids
// bytewise, the same as bytes.Compare.
func newestFirst(posts []dbtest.Post) []uuid.UUID {
	sorted := slices.Clone(posts)
	slices.SortFunc(sorted, func(a, b dbtest.Post) int {
		if c := b.PublishedAt.Compare(*a.PublishedAt); c != 0 {
			return c
		}
		return bytes.Compare(b.ID[:], a.ID[:])
	})
	ids := make([]uuid.UUID, len(sorted))
	for i, p := range sorted {
		ids[i] = p.ID
	}
	return ids
}

func ids(rows []listed) []uuid.UUID {
	out := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

// tieFixture inserts n posts at distinct times plus a group of posts sharing
// one published_at, placed so the group straddles the boundary between pages.
// Every post gets opts (for example a tag). It returns the inserted posts.
func tieFixture(t *testing.T, conn db.DBTX, opts ...dbtest.PostOption) []dbtest.Post {
	t.Helper()

	now := dbtest.Now(t, conn).Truncate(time.Microsecond)
	var posts []dbtest.Post
	insert := func(at time.Time) {
		o := append([]dbtest.PostOption{dbtest.PublishedAt(at)}, opts...)
		posts = append(posts, dbtest.InsertPost(t, conn, o...))
	}

	// With page size 10: positions 1-8 are distinct, 9-13 share a time (so the
	// tie group crosses from page 1 into page 2), then 14-25 are distinct.
	for i := 1; i <= 8; i++ {
		insert(now.Add(-time.Duration(i) * time.Hour))
	}
	tie := now.Add(-9 * time.Hour)
	for range 5 {
		insert(tie)
	}
	for i := 14; i <= 25; i++ {
		insert(now.Add(-time.Duration(i) * time.Hour))
	}
	return posts
}
