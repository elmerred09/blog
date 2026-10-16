package slug_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/slug"
)

func TestMake(t *testing.T) {
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
			got, err := slug.Make(tc.in)
			if err != nil {
				t.Fatalf("Make(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("Make(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if !slug.Valid(got) {
				t.Errorf("Make(%q) = %q, which Valid rejects", tc.in, got)
			}
		})
	}
}

func TestMakeRejectsEmptyResult(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", "   ", "!!!", "---", "日本語"} {
		t.Run(in, func(t *testing.T) {
			t.Parallel()
			if got, err := slug.Make(in); err == nil {
				t.Errorf("Make(%q) = %q, want an error (nothing slug-worthy left)", in, got)
			}
		})
	}
}

func TestValid(t *testing.T) {
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
			if got := slug.Valid(tc.in); got != tc.want {
				t.Errorf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
