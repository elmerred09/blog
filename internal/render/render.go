// Package render turns post markdown into HTML and writes the matching
// highlighting stylesheet.
package render

import (
	"bytes"
	"fmt"
	"io"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// Chroma themes for the stylesheet. Render's HTML only carries class names,
// so changing these needs a new stylesheet (make chroma-css), not a re-import.
const (
	lightStyle = "github"
	darkStyle  = "github-dark"
)

type Renderer struct{ md goldmark.Markdown }

func New() *Renderer {
	return &Renderer{md: goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // tables, strikethrough, autolinks, task lists
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(formatOptions()...),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()), // <h2 id="...">, for anchor links
	)}
}

// Shared by Render and WriteCSS, so the stylesheet's selectors match the HTML.
func formatOptions() []chromahtml.Option {
	return []chromahtml.Option{chromahtml.WithClasses(true)}
}

func (r *Renderer) Render(src []byte) (string, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(src, &buf); err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}
	return buf.String(), nil
}

// WriteCSS writes the stylesheet for the classes Render emits: the light and
// dark themes, each inside its own prefers-color-scheme block. The themes
// style different sets of classes, so an unscoped light theme would leak its
// colors into dark mode wherever the dark theme sets none.
func WriteCSS(w io.Writer) error {
	light, err := style(lightStyle)
	if err != nil {
		return err
	}
	dark, err := style(darkStyle)
	if err != nil {
		return err
	}

	f := chromahtml.New(formatOptions()...)
	var buf bytes.Buffer
	buf.WriteString("@media (prefers-color-scheme: light) {\n")
	if err := f.WriteCSS(&buf, light); err != nil {
		return fmt.Errorf("write %s css: %w", lightStyle, err)
	}
	buf.WriteString("}\n\n@media (prefers-color-scheme: dark) {\n")
	if err := f.WriteCSS(&buf, dark); err != nil {
		return fmt.Errorf("write %s css: %w", darkStyle, err)
	}
	buf.WriteString("}\n")

	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("write css: %w", err)
	}
	return nil
}

// styles.Get falls back to a default style for unknown names; this fails instead.
func style(name string) (*chroma.Style, error) {
	s, ok := styles.Registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown chroma style %q", name)
	}
	return s, nil
}
