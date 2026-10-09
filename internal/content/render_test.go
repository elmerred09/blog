package content_test

import (
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/content"
)

func TestRender(t *testing.T) {
	t.Parallel()

	r := content.NewRenderer()

	tests := []struct {
		name    string
		md      string
		want    []string // substrings the HTML must contain
		notWant []string // substrings it must not contain
	}{
		{
			name: "fenced code is highlighted with classes, not inline styles",
			md:   "```go\nfunc main() {}\n```\n",
			want: []string{`class="chroma"`, `<span class="kd">func</span>`},
			notWant: []string{
				`style=`,
			},
		},
		{
			name:    "raw HTML block is omitted",
			md:      "<script>alert(1)</script>\n",
			want:    []string{"raw HTML omitted"},
			notWant: []string{"<script>"},
		},
		{
			name:    "inline raw HTML is omitted",
			md:      "hello <b onclick=\"x()\">bold</b>\n",
			notWant: []string{"<b", "onclick"},
		},
		{
			name:    "javascript links are neutralized",
			md:      "[click](javascript:alert(1))\n",
			notWant: []string{"javascript:"},
		},
		{
			name: "headings get ids for anchor links",
			md:   "## Hello World\n",
			want: []string{`<h2 id="hello-world">Hello World</h2>`},
		},
		{
			name: "GFM tables",
			md:   "| a | b |\n|---|---|\n| 1 | 2 |\n",
			want: []string{"<table>", "<td>1</td>"},
		},
		{
			name: "GFM strikethrough",
			md:   "~~gone~~\n",
			want: []string{"<del>gone</del>"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			html, err := r.Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			for _, s := range tc.want {
				if !strings.Contains(html, s) {
					t.Errorf("output missing %q\ngot:\n%s", s, html)
				}
			}
			for _, s := range tc.notWant {
				if strings.Contains(html, s) {
					t.Errorf("output contains %q\ngot:\n%s", s, html)
				}
			}
		})
	}
}
