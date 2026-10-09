package content_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/content"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"single word", "Postgres", "postgres"},
		{"spaces become dashes", "Postgres Internals", "postgres-internals"},
		{"surrounding space trimmed", "  Go  ", "go"},
		{"punctuation becomes a dash", "Node.js", "node-js"},
		{"underscores become dashes", "snake_case", "snake-case"},
		{"digits kept", "Go 1.27", "go-1-27"},
		{"existing dashes kept", "full-text search", "full-text-search"},
		{"runs collapse to one dash", "a  --  b", "a-b"},
		{"no leading or trailing dash", "--Edge Cases!--", "edge-cases"},
		{"accents folded", "Café Über", "cafe-uber"},
		{"already a slug", "keyset-pagination", "keyset-pagination"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.Slugify(tc.in)
			if err != nil {
				t.Fatalf("Slugify(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if !content.ValidSlug(got) {
				t.Errorf("Slugify(%q) = %q, which ValidSlug rejects", tc.in, got)
			}
		})
	}
}

func TestSlugifyRejectsEmptyResult(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", "   ", "!!!", "---", "日本語"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got, err := content.Slugify(in); err == nil {
				t.Errorf("Slugify(%q) = %q, want an error (nothing slug-worthy left)", in, got)
			}
		})
	}
}

func TestValidSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want bool
	}{
		{"go", true},
		{"postgres-internals", true},
		{"go-1-27", true},
		{"a1", true},
		{"", false},
		{"Go", false},
		{"-go", false},
		{"go-", false},
		{"go--x", false},
		{"go_x", false},
		{"go x", false},
		{"café", false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := content.ValidSlug(tc.in); got != tc.want {
				t.Errorf("ValidSlug(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
