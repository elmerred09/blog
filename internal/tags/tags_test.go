package tags_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/tags"
)

func TestFromNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		names []string
		want  []tag
	}{
		{"none", nil, nil},
		{"keeps order", []string{"Postgres", "Go"}, []tag{{"postgres", "Postgres"}, {"go", "Go"}}},
		{"trims the name", []string{"  Full-Text Search "}, []tag{{"full-text-search", "Full-Text Search"}}},
		{
			"names sharing a slug collapse to the first",
			[]string{"Postgres", "postgres", "Go", "POSTGRES"},
			[]tag{{"postgres", "Postgres"}, {"go", "Go"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tags.FromNames(tc.names)
			if err != nil {
				t.Fatalf("FromNames(%q): %v", tc.names, err)
			}
			if want := toTags(tc.want); !slices.Equal(got, want) {
				t.Errorf("FromNames(%q) = %v, want %v", tc.names, got, want)
			}
		})
	}
}

func TestFromNamesReportsEveryBadName(t *testing.T) {
	t.Parallel()

	got, err := tags.FromNames([]string{"Go", "!!!", "日本語"})
	if err == nil {
		t.Fatalf("FromNames = %v, want an error", got)
	}
	for _, bad := range []string{"!!!", "日本語"} {
		if !strings.Contains(err.Error(), bad) {
			t.Errorf("error %q doesn't mention %q", err, bad)
		}
	}
}

// tag keeps the test tables short; tags.Tag has named fields.
type tag struct{ slug, name string }

func toTags(ts []tag) []tags.Tag {
	if ts == nil {
		return nil
	}
	out := make([]tags.Tag, len(ts))
	for i, t := range ts {
		out[i] = tags.Tag{Slug: t.slug, Name: t.name}
	}
	return out
}
