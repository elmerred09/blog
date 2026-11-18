package render

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// anchorClass marks the link wrapping a heading's text; the web styles it.
const anchorClass = "heading-anchor"

// headingAnchors wraps each heading's text in a link to the heading's own id,
// so a reader can copy a link to that section. A heading that already holds
// a link is left alone: links can't nest.
type headingAnchors struct{}

func (headingAnchors) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		attr, _ := h.AttributeString("id")
		if id, _ := attr.([]byte); len(id) > 0 && h.HasChildren() && !hasLink(h) {
			wrap(h, id)
		}
		return ast.WalkSkipChildren, nil
	})
}

func hasLink(n ast.Node) bool {
	found := false
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		switch c.(type) {
		case *ast.Link, *ast.AutoLink:
			found = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

func wrap(h *ast.Heading, id []byte) {
	link := ast.NewLink()
	link.Destination = append([]byte("#"), id...)
	link.SetAttributeString("class", []byte(anchorClass))
	for c := h.FirstChild(); c != nil; {
		next := c.NextSibling()
		link.AppendChild(link, c)
		c = next
	}
	h.AppendChild(h, link)
}
