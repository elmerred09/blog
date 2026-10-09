package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// fsPaths converts command-line paths into the clean, slash-separated paths
// relative to root that content.Import expects for an fs.FS backed by root.
// Relative arguments are taken relative to root; paths outside root are errors.
func fsPaths(root string, args []string) ([]string, error) {
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		p := arg
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", arg, err)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s is outside %s", arg, root)
		}
		paths = append(paths, filepath.ToSlash(rel))
	}
	return paths, nil
}
