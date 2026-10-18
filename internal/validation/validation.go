// Package validation is the one error type for input a service rejects. Each
// Problem names a field, so a form can show its message beside the input, and
// the HTTP layer maps every service's validation failures the same way.
package validation

import "strings"

// Problem is one field that failed. Field is the input's name as the client
// sends it (title, body_md, q). Message is safe to show the user.
type Problem struct {
	Field   string
	Message string
}

// Error is rejected input. Problems lists every failure, in a stable order.
type Error struct {
	Problems []Problem
}

// New returns an *Error holding problems, or nil when there are none, so a
// validator can end with "return validation.New(problems...)".
func New(problems ...Problem) error {
	if len(problems) == 0 {
		return nil
	}
	return &Error{Problems: problems}
}

// Error lists every problem as "field: message", for logs.
func (e *Error) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.Field + ": " + p.Message
	}
	return "invalid input: " + strings.Join(parts, "; ")
}

// Summary joins the messages, without field names, for a one-line message a
// client can show when it doesn't place them beside fields.
func (e *Error) Summary() string {
	msgs := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		msgs[i] = p.Message
	}
	return strings.Join(msgs, "; ")
}
