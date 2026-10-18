package admin_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/render"
	"github.com/elmerred09/blog/internal/tags"
	"github.com/elmerred09/blog/internal/validation"
)

func newService(t *testing.T) (*pgxpool.Pool, *admin.Service) {
	t.Helper()
	pool, _ := env.NewDB(t)
	return pool, admin.NewService(pool, render.New())
}

func input(slug string) admin.Input {
	return admin.Input{
		Slug:    slug,
		Title:   "Title of " + slug,
		Summary: "Summary of " + slug,
		BodyMD:  "# " + slug,
	}
}

// loaded turns a post the service returned into the save that would store it
// unchanged. Tag names are the display names, so an existing tag keeps its name.
func loaded(p admin.Post) admin.Update {
	names := make([]string, len(p.Tags))
	for i, tag := range p.Tags {
		names[i] = tag.Name
	}
	return admin.Update{
		ID:                p.ID,
		ExpectedUpdatedAt: p.UpdatedAt,
		Slug:              p.Slug,
		Title:             p.Title,
		Summary:           p.Summary,
		BodyMD:            p.BodyMD,
		PublishedAt:       p.PublishedAt,
		TagNames:          names,
	}
}

func TestCreateAndGet(t *testing.T) {
	t.Parallel()
	_, svc := newService(t)

	in := input("hello")
	in.TagNames = []string{"Go", "Postgres"}
	created, err := svc.Create(t.Context(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Slug != in.Slug || got.Title != in.Title || got.Summary != in.Summary || got.BodyMD != in.BodyMD {
		t.Errorf("post = %+v, want the created fields", got)
	}
	if got.PublishedAt != nil || got.State != admin.Draft {
		t.Errorf("published_at = %v, state = %q, want a draft", got.PublishedAt, got.State)
	}
	if !got.UpdatedAt.Equal(created.UpdatedAt) || got.UpdatedAt.IsZero() {
		t.Errorf("updated_at = %v, created returned %v", got.UpdatedAt, created.UpdatedAt)
	}
	want := []tags.Tag{{Slug: "go", Name: "Go"}, {Slug: "postgres", Name: "Postgres"}}
	if !slices.Equal(got.Tags, want) {
		t.Errorf("tags = %+v, want %+v", got.Tags, want)
	}
	if got.Tags == nil {
		t.Errorf("tags is nil")
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	_, svc := newService(t)

	_, err := svc.Create(t.Context(), admin.Input{Slug: "Hello"})
	var invalid *validation.Error
	if !errors.As(err, &invalid) || len(invalid.Problems) == 0 || invalid.Problems[0].Field != "title" {
		t.Errorf("err = %v, want a *validation.Error whose first problem is the title", err)
	}
}

func TestCreateSlugTaken(t *testing.T) {
	t.Parallel()
	_, svc := newService(t)

	if _, err := svc.Create(t.Context(), input("hello")); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.Create(t.Context(), input("hello")); !errors.Is(err, admin.ErrSlugTaken) {
		t.Errorf("second create: err = %v, want ErrSlugTaken", err)
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	_, svc := newService(t)
	ctx := t.Context()

	created, err := svc.Create(ctx, input("hello"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	t.Run("a stale token conflicts and keeps the post", func(t *testing.T) {
		stale := loaded(created)
		stale.Title = "Stale"
		stale.ExpectedUpdatedAt = created.UpdatedAt.Add(-time.Second)
		if _, err := svc.Update(ctx, stale); !errors.Is(err, admin.ErrConflict) {
			t.Fatalf("Update: err = %v, want ErrConflict", err)
		}
		got, err := svc.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.Title != created.Title || !got.UpdatedAt.Equal(created.UpdatedAt) {
			t.Errorf("post changed: title %q, updated_at %v", got.Title, got.UpdatedAt)
		}
	})

	saved := created
	t.Run("two saves in a row each take the token the last one returned", func(t *testing.T) {
		first := loaded(saved)
		first.Title = "First"
		saved, err = svc.Update(ctx, first)
		if err != nil {
			t.Fatalf("first save: %v", err)
		}
		if !saved.UpdatedAt.After(created.UpdatedAt) || saved.Title != "First" {
			t.Fatalf("first save = title %q, updated_at %v", saved.Title, saved.UpdatedAt)
		}

		second := loaded(saved)
		second.Title = "Second"
		saved, err = svc.Update(ctx, second)
		if err != nil {
			t.Fatalf("second save: %v", err)
		}
		if !saved.UpdatedAt.After(first.ExpectedUpdatedAt) || saved.Title != "Second" {
			t.Errorf("second save = title %q, updated_at %v", saved.Title, saved.UpdatedAt)
		}
	})

	t.Run("a taken slug", func(t *testing.T) {
		other, err := svc.Create(ctx, input("other"))
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		clash := loaded(saved)
		clash.Slug = other.Slug
		if _, err := svc.Update(ctx, clash); !errors.Is(err, admin.ErrSlugTaken) {
			t.Errorf("err = %v, want ErrSlugTaken", err)
		}
	})
}

func TestUpdateNotFound(t *testing.T) {
	t.Parallel()
	pool, svc := newService(t)

	deleted := dbtest.InsertPost(t, pool, dbtest.Deleted())
	var token time.Time
	err := pool.QueryRow(t.Context(), `SELECT updated_at FROM posts WHERE id = $1`, deleted.ID).Scan(&token)
	if err != nil {
		t.Fatalf("read updated_at: %v", err)
	}

	tests := []struct {
		name string
		u    admin.Update
	}{
		{"unknown id", admin.Update{ID: uuid.New(), ExpectedUpdatedAt: time.Now(), Input: input("missing")}},
		{"deleted post", admin.Update{ID: deleted.ID, ExpectedUpdatedAt: token, Input: input("deleted-edit")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Update(t.Context(), tc.u); !errors.Is(err, admin.ErrNotFound) {
				t.Errorf("err = %v, want ErrNotFound", err)
			}
		})
	}

	if _, err := svc.Get(t.Context(), deleted.ID); !errors.Is(err, admin.ErrNotFound) {
		t.Errorf("Get(deleted) err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(t.Context(), uuid.New()); !errors.Is(err, admin.ErrNotFound) {
		t.Errorf("Get(unknown) err = %v, want ErrNotFound", err)
	}
}

func TestPublishChangesPublicVisibility(t *testing.T) {
	t.Parallel()
	pool, svc := newService(t)
	ctx := t.Context()
	public := post.NewService(pool)

	created, err := svc.Create(ctx, input("visible"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := public.Get(ctx, "visible"); !errors.Is(err, post.ErrNotFound) {
		t.Fatalf("draft is public: %v", err)
	}

	save := func(at *time.Time) {
		t.Helper()
		u := loaded(created)
		u.PublishedAt = at
		created, err = svc.Update(ctx, u)
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	past := time.Now().Add(-time.Hour)
	save(&past)
	if created.State != admin.Published {
		t.Errorf("state = %q, want published", created.State)
	}
	if _, err := public.Get(ctx, "visible"); err != nil {
		t.Fatalf("published post: %v", err)
	}

	future := time.Now().Add(time.Hour)
	save(&future)
	if created.State != admin.Scheduled {
		t.Errorf("state = %q, want scheduled", created.State)
	}
	if _, err := public.Get(ctx, "visible"); !errors.Is(err, post.ErrNotFound) {
		t.Errorf("scheduled post is public: %v", err)
	}

	save(nil)
	if created.State != admin.Draft {
		t.Errorf("state = %q, want draft", created.State)
	}
	if _, err := public.Get(ctx, "visible"); !errors.Is(err, post.ErrNotFound) {
		t.Errorf("unpublished post is public: %v", err)
	}
}

func TestTagSyncOnEdit(t *testing.T) {
	t.Parallel()
	pool, svc := newService(t)
	ctx := t.Context()

	// An existing tag keeps the name it was created with.
	dbtest.InsertTag(t, pool, "go")
	in := input("tagged")
	in.TagNames = []string{"Go", "Postgres"}
	created, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if want := []tags.Tag{{Slug: "go", Name: "go"}, {Slug: "postgres", Name: "Postgres"}}; !slices.Equal(created.Tags, want) {
		t.Fatalf("tags = %+v, want %+v (the existing go tag keeps its name)", created.Tags, want)
	}

	u := loaded(created)
	u.TagNames = []string{"Go", "Search"}
	saved, err := svc.Update(ctx, u)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if want := []tags.Tag{{Slug: "go", Name: "go"}, {Slug: "search", Name: "Search"}}; !slices.Equal(saved.Tags, want) {
		t.Errorf("tags = %+v, want %+v", saved.Tags, want)
	}

	u = loaded(saved)
	u.TagNames = nil
	saved, err = svc.Update(ctx, u)
	if err != nil {
		t.Fatalf("clear tags: %v", err)
	}
	if saved.Tags == nil || len(saved.Tags) != 0 {
		t.Errorf("tags = %+v, want none", saved.Tags)
	}
}

func TestPreviewMatchesSavedHTML(t *testing.T) {
	t.Parallel()
	pool, svc := newService(t)

	const md = "# Hello\n\n```go\nfmt.Println(1)\n```\n"
	in := input("rendered")
	in.BodyMD = md
	created, err := svc.Create(t.Context(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	preview, err := svc.Preview(md)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	var stored string
	err = pool.QueryRow(t.Context(), `SELECT body_html FROM posts WHERE id = $1`, created.ID).Scan(&stored)
	if err != nil {
		t.Fatalf("read body_html: %v", err)
	}
	if preview != stored {
		t.Errorf("preview = %q, stored = %q", preview, stored)
	}
}

func TestList(t *testing.T) {
	t.Parallel()
	pool, svc := newService(t)

	var want []uuid.UUID
	for i := range 12 {
		var opt dbtest.PostOption
		switch i % 3 {
		case 0:
			opt = dbtest.PublishedAgo(time.Duration(i+1) * time.Hour)
		case 1:
			opt = dbtest.Draft()
		default:
			opt = dbtest.ScheduledIn(time.Duration(i+1) * time.Hour)
		}
		want = append(want, dbtest.InsertPost(t, pool, opt).ID)
	}
	dbtest.InsertPost(t, pool, dbtest.Deleted())
	slices.SortFunc(want, func(a, b uuid.UUID) int { return bytes.Compare(b[:], a[:]) })

	const limit = 5
	var (
		got   []uuid.UUID
		after *uuid.UUID
		pages int
	)
	for pages = 1; pages <= 10; pages++ {
		page, err := svc.List(t.Context(), admin.ListParams{After: after, Limit: limit})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if page.Items == nil {
			t.Fatal("Items is nil")
		}
		for _, item := range page.Items {
			got = append(got, item.ID)
			if item.State == "" {
				t.Errorf("post %s has an empty state", item.ID)
			}
		}
		if page.Next == nil {
			break
		}
		if len(page.Items) != limit {
			t.Fatalf("page %d has %d items and a Next", pages, len(page.Items))
		}
		after = page.Next
	}

	if !slices.Equal(got, want) {
		t.Errorf("ids = %v,\nwant %v", got, want)
	}
	if pages != 3 {
		t.Errorf("pages = %d, want 3", pages)
	}
}

func TestListLimit(t *testing.T) {
	t.Parallel()
	_, svc := newService(t)

	for _, limit := range []int{0, admin.MaxLimit + 1} {
		if _, err := svc.List(t.Context(), admin.ListParams{Limit: limit}); !errors.Is(err, admin.ErrInvalidLimit) {
			t.Errorf("List(limit %d) err = %v, want ErrInvalidLimit", limit, err)
		}
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()
	_, svc := newService(t)

	page, err := svc.List(t.Context(), admin.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Items == nil || len(page.Items) != 0 || page.Next != nil {
		t.Errorf("page = %+v, want empty Items and no Next", page)
	}
}
