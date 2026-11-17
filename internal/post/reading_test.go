package post_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/post"
)

func TestReadingMinutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		words int32
		want  int
	}{
		{"empty body still reads as a minute", 0, 1},
		{"one word", 1, 1},
		{"exactly one minute", 230, 1},
		{"one word over rounds up", 231, 2},
		{"ten minutes", 2300, 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := post.ReadingMinutes(tc.words); got != tc.want {
				t.Errorf("ReadingMinutes(%d) = %d, want %d", tc.words, got, tc.want)
			}
		})
	}
}
