package search

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/elmerred09/blog/internal/post"
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
	// is an empty, non-nil slice. An invalid q is a *ValidationError.
	Search(ctx context.Context, q Query) ([]Result, error)
}

type Query struct {
	// Text is the user's input, as typed.
	Text  string
	Limit int
}

// ValidationError is an invalid Query. Message is safe to show the client.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "invalid search query: " + e.Message
}

func (q Query) Validate() error {
	text := strings.TrimSpace(q.Text)
	if text == "" {
		return &ValidationError{Message: "q is required"}
	}
	if utf8.RuneCountInString(text) > MaxQueryLen {
		return &ValidationError{Message: fmt.Sprintf("q must be at most %d characters", MaxQueryLen)}
	}
	if q.Limit < 1 || q.Limit > MaxLimit {
		return &ValidationError{Message: fmt.Sprintf("limit must be an integer from 1 to %d", MaxLimit)}
	}
	return nil
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
