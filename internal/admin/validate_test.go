package admin

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/elmerred09/blog/internal/render"
	"github.com/elmerred09/blog/internal/validation"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	ok := Input{Slug: "hello", Title: "Hello", Summary: "A post.", BodyMD: "# Hello"}

	tests := []struct {
		name string
		in   Input
		want []validation.Problem
	}{
		{"accepts a complete draft", ok, nil},
		{"trims the title and summary", Input{Slug: "hello", Title: "  Hello  ", Summary: "  A post.  "}, nil},
		{"title is required", Input{Slug: "hello", Title: "  ", Summary: "A post."}, []validation.Problem{{Field: "title", Message: "title is required"}}},
		{"summary is required", Input{Slug: "hello", Title: "Hello", Summary: ""}, []validation.Problem{{Field: "summary", Message: "summary is required"}}},
		{
			"slug must be valid",
			Input{Slug: "Hello World", Title: "Hello", Summary: "A post."},
			[]validation.Problem{{Field: "slug", Message: `slug "Hello World" must be lowercase letters, digits, and single dashes`}},
		},
		{
			"empty slug",
			Input{Title: "Hello", Summary: "A post."},
			[]validation.Problem{{Field: "slug", Message: `slug "" must be lowercase letters, digits, and single dashes`}},
		},
		{
			"body over the limit",
			Input{Slug: "hello", Title: "Hello", Summary: "A post.", BodyMD: strings.Repeat("a", MaxBodyBytes+1)},
			[]validation.Problem{{Field: "body_md", Message: bodyTooBig(MaxBodyBytes + 1)}},
		},
		{
			"each bad tag is its own problem",
			Input{Slug: "hello", Title: "Hello", Summary: "A post.", TagNames: []string{"???", "!!!"}},
			[]validation.Problem{
				{Field: "tags", Message: `tag: slugify "???": no letters or digits`},
				{Field: "tags", Message: `tag: slugify "!!!": no letters or digits`},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := validate(tc.in)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("validate: %v", err)
				}
				if got.title != strings.TrimSpace(tc.in.Title) || got.summary != strings.TrimSpace(tc.in.Summary) {
					t.Errorf("stored title %q, summary %q", got.title, got.summary)
				}
				return
			}
			var invalid *validation.Error
			if !errors.As(err, &invalid) || !slices.Equal(invalid.Problems, tc.want) {
				t.Errorf("err = %v, want problems %v", err, tc.want)
			}
		})
	}

	t.Run("reports every problem", func(t *testing.T) {
		t.Parallel()
		_, err := validate(Input{})
		var invalid *validation.Error
		if !errors.As(err, &invalid) {
			t.Fatalf("err = %v, want a *validation.Error", err)
		}
		want := []validation.Problem{
			{Field: "title", Message: "title is required"},
			{Field: "summary", Message: "summary is required"},
			{Field: "slug", Message: `slug "" must be lowercase letters, digits, and single dashes`},
		}
		if !slices.Equal(invalid.Problems, want) {
			t.Errorf("problems = %v, want %v", invalid.Problems, want)
		}
	})

	t.Run("a body at the limit is accepted", func(t *testing.T) {
		t.Parallel()
		in := ok
		in.BodyMD = strings.Repeat("a", MaxBodyBytes)
		if _, err := validate(in); err != nil {
			t.Fatalf("validate: %v", err)
		}
	})

	t.Run("duplicate tag names collapse", func(t *testing.T) {
		t.Parallel()
		in := ok
		in.TagNames = []string{" Go ", "go", "Postgres"}
		got, err := validate(in)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if len(got.tags) != 2 || got.tags[0].Slug != "go" || got.tags[0].Name != "Go" || got.tags[1].Slug != "postgres" {
			t.Errorf("tags = %+v, want go (named Go) then postgres", got.tags)
		}
	})
}

func TestStateOf(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Second)
	future := now.Add(time.Second)

	tests := []struct {
		name string
		at   *time.Time
		want State
	}{
		{"nil is a draft", nil, Draft},
		{"past is published", &past, Published},
		{"exactly now is published", &now, Published},
		{"future is scheduled", &future, Scheduled},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := StateOf(tc.at, now); got != tc.want {
				t.Errorf("StateOf = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPreviewRejectsAnOversizedBody(t *testing.T) {
	t.Parallel()
	_, err := NewService(nil, render.New()).Preview(strings.Repeat("a", MaxBodyBytes+1))
	var invalid *validation.Error
	want := validation.Problem{Field: "body_md", Message: bodyTooBig(MaxBodyBytes + 1)}
	if !errors.As(err, &invalid) || len(invalid.Problems) != 1 || invalid.Problems[0] != want {
		t.Errorf("err = %v, want a *validation.Error with %v", err, want)
	}
}
