package validation

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	if err := New(); err != nil {
		t.Errorf("New() = %v, want nil", err)
	}

	err := fmt.Errorf("save: %w", New(
		Problem{Field: "title", Message: "title is required"},
		Problem{Field: "slug", Message: "slug is invalid"},
	))
	var invalid *Error
	if !errors.As(err, &invalid) {
		t.Fatalf("errors.As failed on %v", err)
	}
	if got, want := invalid.Error(), "invalid input: title: title is required; slug: slug is invalid"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if got, want := invalid.Summary(), "title is required; slug is invalid"; got != want {
		t.Errorf("Summary() = %q, want %q", got, want)
	}
}
