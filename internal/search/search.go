package search

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/validation"
)

const (
	// DefaultLimit is the result count callers should use when none is given.
	DefaultLimit = 20
	// MaxLimit is the largest result count a Query may ask for.
	MaxLimit = 50
	// MaxQueryLen is the longest query text, in characters, after trimming.
	MaxQueryLen = 200
)

// matchStart and matchStop are the ts_headline StartSel and StopSel in the
// SearchPosts query. They're Unicode private-use characters, so they don't
// occur in post text, and the snippet needs no HTML parsing or escaping.
const (
	matchStart = "\uE000"
	matchStop  = "\uE001"
)

type Searcher interface {
	// Search returns the best matches first, at most q.Limit of them. No match
	// is an empty, non-nil slice. An invalid q is a *validation.Error.
	Search(ctx context.Context, q Query) ([]Result, error)
}

type Query struct {
	// Text is the user's input, as typed.
	Text  string
	Limit int
}

// LimitMessage is the problem message for a limit outside 1 to MaxLimit, or
// one that isn't an integer at all.
var LimitMessage = fmt.Sprintf("limit must be an integer from 1 to %d", MaxLimit)

// Validate returns a *validation.Error with a problem for q, limit, or both.
func (q Query) Validate() error {
	var problems []validation.Problem
	text := strings.TrimSpace(q.Text)
	switch {
	case text == "":
		problems = append(problems, validation.Problem{Field: "q", Message: "q is required"})
	case utf8.RuneCountInString(text) > MaxQueryLen:
		problems = append(problems, validation.Problem{
			Field:   "q",
			Message: fmt.Sprintf("q must be at most %d characters", MaxQueryLen),
		})
	}
	if q.Limit < 1 || q.Limit > MaxLimit {
		problems = append(problems, validation.Problem{Field: "limit", Message: LimitMessage})
	}
	return validation.New(problems...)
}

type Result struct {
	post.ListItem
	// Snippet is an excerpt around the matches, never nil.
	Snippet []Fragment
}

// Fragment is plain text, not HTML. Match marks a matched word.
type Fragment struct {
	Text  string
	Match bool
}

// splitSnippet turns a headline with matchStart/matchStop markers into
// fragments. Stray markers are dropped, an unclosed match runs to the end, and
// neighbouring fragments of the same kind are merged.
func splitSnippet(headline string) []Fragment {
	frags := []Fragment{}
	add := func(text string, match bool) {
		if text == "" {
			return
		}
		if n := len(frags); n > 0 && frags[n-1].Match == match {
			frags[n-1].Text += text
			return
		}
		frags = append(frags, Fragment{Text: text, Match: match})
	}

	match := false
	for headline != "" {
		i := strings.IndexAny(headline, matchStart+matchStop)
		if i < 0 {
			add(headline, match)
			break
		}
		add(headline[:i], match)
		marker, size := utf8.DecodeRuneInString(headline[i:])
		match = string(marker) == matchStart
		headline = headline[i+size:]
	}
	return frags
}
