package main

import (
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/content"
)

func TestPrintSummary(t *testing.T) {
	t.Parallel()

	s := content.Summary{
		Files: []content.FileResult{
			{Path: "content/a-long-name.md", Slug: "a-long-name", Outcome: content.Inserted},
			{Path: "content/b.md", Slug: "b", Outcome: content.Unchanged},
		},
		TagsCreated:     2,
		TagLinksAdded:   3,
		TagLinksRemoved: 1,
	}

	var out strings.Builder
	if err := printSummary(&out, s); err != nil {
		t.Fatalf("printSummary: %v", err)
	}

	want := `PATH                    SLUG         OUTCOME
content/a-long-name.md  a-long-name  inserted
content/b.md            b            unchanged

1 inserted, 0 updated, 1 unchanged; 2 tags created; 3 tag links added, 1 removed
`
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
}
