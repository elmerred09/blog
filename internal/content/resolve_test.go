package content_test

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/elmerred09/blog/internal/content"
)

func resolveFS() fstest.MapFS {
	file := &fstest.MapFile{Data: []byte("x")}
	return fstest.MapFS{
		"a.md":            file,
		"notes.txt":       file,
		"dir/b.md":        file,
		"dir/image.png":   file,
		"dir/sub/c.md":    file,
		"other/d.md":      file,
		"assets/logo.svg": file,
		"empty":           &fstest.MapFile{Mode: fs.ModeDir},
	}
}

func TestResolvePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{"single file", []string{"a.md"}, []string{"a.md"}},
		{"directory is recursive and skips non-markdown", []string{"dir"}, []string{"dir/b.md", "dir/sub/c.md"}},
		{"root directory", []string{"."}, []string{"a.md", "dir/b.md", "dir/sub/c.md", "other/d.md"}},
		{"result is sorted", []string{"other", "a.md"}, []string{"a.md", "other/d.md"}},
		{
			"a file named directly and through its directory appears once",
			[]string{"dir/b.md", "dir", "dir/b.md"},
			[]string{"dir/b.md", "dir/sub/c.md"},
		},
		{"nested directory inside a named one", []string{"dir", "dir/sub"}, []string{"dir/b.md", "dir/sub/c.md"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.ResolvePaths(resolveFS(), tc.paths)
			if err != nil {
				t.Fatalf("ResolvePaths(%q): %v", tc.paths, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("ResolvePaths(%q) = %q, want %q", tc.paths, got, tc.want)
			}
		})
	}
}

func TestResolvePathsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		paths   []string
		wantErr string // substring; the error must name the offending path
	}{
		{"no paths", nil, "no paths"},
		{"missing file", []string{"a.md", "missing.md"}, "missing.md"},
		{"non-markdown file named directly", []string{"notes.txt"}, "notes.txt"},
		{"directory without markdown", []string{"assets"}, "assets"},
		{"empty directory", []string{"empty"}, "empty"},
		{"path escaping the FS", []string{"../secret.md"}, "../secret.md"},
		{"unclean path", []string{"./a.md"}, "./a.md"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.ResolvePaths(resolveFS(), tc.paths)
			if err == nil {
				t.Fatalf("ResolvePaths(%q) = %q, want an error mentioning %q", tc.paths, got, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not mention %q", err, tc.wantErr)
			}
		})
	}
}
