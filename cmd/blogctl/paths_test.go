package main

import (
	"slices"
	"strings"
	"testing"
)

func TestFSPaths(t *testing.T) {
	t.Parallel()

	const root = "/home/me/blog"

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{"relative file", []string{"content/a.md"}, []string{"content/a.md"}},
		{"leading ./ and trailing slash", []string{"./content/"}, []string{"content"}},
		{"doubled slashes", []string{"content//a.md"}, []string{"content/a.md"}},
		{"root itself", []string{"."}, []string{"."}},
		{"absolute inside root", []string{"/home/me/blog/content/a.md"}, []string{"content/a.md"}},
		{"dot-dot that stays inside", []string{"content/../content/a.md"}, []string{"content/a.md"}},
		{"name starting with dots is not a parent", []string{"..drafts/a.md"}, []string{"..drafts/a.md"}},
		{"order kept", []string{"b.md", "a.md"}, []string{"b.md", "a.md"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := fsPaths(root, tc.args)
			if err != nil {
				t.Fatalf("fsPaths(%q): %v", tc.args, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("fsPaths(%q) = %q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func TestFSPathsOutsideRoot(t *testing.T) {
	t.Parallel()

	const root = "/home/me/blog"

	for _, arg := range []string{"../other.md", "..", "/etc/passwd", "/home/me/blog-other/a.md", "content/../../x.md"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()
			got, err := fsPaths(root, []string{"content/a.md", arg})
			if err == nil {
				t.Fatalf("fsPaths accepted %q as %q", arg, got)
			}
			if !strings.Contains(err.Error(), arg) {
				t.Errorf("error %q does not name %q", err, arg)
			}
		})
	}
}
