package content_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/elmerred09/blog/internal/content"
)

func TestParse(t *testing.T) {
	t.Parallel()

	eastern := time.FixedZone("EDT", -4*60*60)

	tests := []struct {
		name string
		path string
		src  string

		wantSlug      string
		wantTitle     string
		wantSummary   string
		wantPublished *time.Time // nil means draft
		wantTags      []content.Tag
		wantBody      string
	}{
		{
			name: "full frontmatter",
			path: "keyset.md",
			src: `---
title: Keyset pagination
summary: Why OFFSET gets slow.
slug: keyset-pagination
published_at: 2026-10-01T09:00:00-04:00
tags: [Postgres, Performance]
---
# Hello
`,
			wantSlug:      "keyset-pagination",
			wantTitle:     "Keyset pagination",
			wantSummary:   "Why OFFSET gets slow.",
			wantPublished: new(time.Date(2026, 10, 1, 9, 0, 0, 0, eastern)),
			wantTags:      []content.Tag{{Slug: "postgres", Name: "Postgres"}, {Slug: "performance", Name: "Performance"}},
			wantBody:      "# Hello\n",
		},
		{
			name:     "slug falls back to the filename",
			path:     "2026/Keyset Pagination.md",
			src:      "---\ntitle: T\nsummary: S\n---\nbody\n",
			wantSlug: "keyset-pagination",
			wantBody: "body\n",
		},
		{
			name:          "bare date is midnight UTC",
			path:          "a.md",
			src:           "---\ntitle: T\nsummary: S\npublished_at: 2026-10-01\n---\n",
			wantSlug:      "a",
			wantPublished: new(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:     "no published_at is a draft",
			path:     "draft.md",
			src:      "---\ntitle: T\nsummary: S\n---\n",
			wantSlug: "draft",
		},
		{
			name:     "tags with the same slug collapse, first name wins",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: S\ntags: [Postgres, postgres, Go, POSTGRES]\n---\n",
			wantSlug: "a",
			wantTags: []content.Tag{{Slug: "postgres", Name: "Postgres"}, {Slug: "go", Name: "Go"}},
		},
		{
			name:     "CRLF line endings",
			path:     "a.md",
			src:      "---\r\ntitle: T\r\nsummary: S\r\n---\r\nline one\r\nline two\r\n",
			wantSlug: "a",
			wantBody: "line one\nline two\n",
		},
		{
			name:     "a --- inside the body is not a delimiter",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: S\n---\nabove\n\n---\n\nbelow\n",
			wantSlug: "a",
			wantBody: "above\n\n---\n\nbelow\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.Parse(tc.path, []byte(tc.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			if got.Path != tc.path {
				t.Errorf("Path = %q, want %q", got.Path, tc.path)
			}
			if got.Slug != tc.wantSlug {
				t.Errorf("Slug = %q, want %q", got.Slug, tc.wantSlug)
			}
			if tc.wantTitle != "" && got.Title != tc.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tc.wantTitle)
			}
			if tc.wantSummary != "" && got.Summary != tc.wantSummary {
				t.Errorf("Summary = %q, want %q", got.Summary, tc.wantSummary)
			}
			switch {
			case tc.wantPublished == nil && got.PublishedAt != nil:
				t.Errorf("PublishedAt = %v, want nil (draft)", *got.PublishedAt)
			case tc.wantPublished != nil && got.PublishedAt == nil:
				t.Errorf("PublishedAt = nil, want %v", *tc.wantPublished)
			case tc.wantPublished != nil && !got.PublishedAt.Equal(*tc.wantPublished):
				t.Errorf("PublishedAt = %v, want %v", *got.PublishedAt, *tc.wantPublished)
			}
			if !slices.Equal(got.Tags, tc.wantTags) {
				t.Errorf("Tags = %+v, want %+v", got.Tags, tc.wantTags)
			}
			if got.BodyMD != tc.wantBody {
				t.Errorf("BodyMD = %q, want %q", got.BodyMD, tc.wantBody)
			}
		})
	}
}

// Errors are checked by substring: each message must name what is wrong, so
// the author can fix the file without reading the parser.
func TestParseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		src      string
		wantErrs []string
	}{
		{
			name:     "no frontmatter",
			path:     "a.md",
			src:      "# Just markdown\n",
			wantErrs: []string{"frontmatter"},
		},
		{
			name:     "unclosed frontmatter",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: S\n# body\n",
			wantErrs: []string{"frontmatter"},
		},
		{
			name:     "missing title",
			path:     "a.md",
			src:      "---\nsummary: S\n---\n",
			wantErrs: []string{"title"},
		},
		{
			name:     "blank summary",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: \"  \"\n---\n",
			wantErrs: []string{"summary"},
		},
		{
			name:     "every problem is reported, not just the first",
			path:     "a.md",
			src:      "---\nslug: Bad_Slug\n---\n",
			wantErrs: []string{"title", "summary", "slug"},
		},
		{
			name:     "unknown key (typo) is rejected",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: S\npublised_at: 2026-10-01\n---\n",
			wantErrs: []string{"publised_at"},
		},
		{
			name:     "malformed YAML",
			path:     "a.md",
			src:      "---\ntitle: [unclosed\nsummary: S\n---\n",
			wantErrs: []string{"yaml"},
		},
		{
			name:     "invalid frontmatter slug is an error, not fixed",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: S\nslug: Keyset_Pagination\n---\n",
			wantErrs: []string{"Keyset_Pagination"},
		},
		{
			name:     "filename with nothing slug-worthy",
			path:     "!!!.md",
			src:      "---\ntitle: T\nsummary: S\n---\n",
			wantErrs: []string{"slug"},
		},
		{
			name:     "tag with nothing slug-worthy",
			path:     "a.md",
			src:      "---\ntitle: T\nsummary: S\ntags: [Go, \"!!!\"]\n---\n",
			wantErrs: []string{"!!!"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.Parse(tc.path, []byte(tc.src))
			if err == nil {
				t.Fatalf("Parse succeeded with %+v, want an error mentioning %q", got, tc.wantErrs)
			}
			for _, want := range tc.wantErrs {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}
