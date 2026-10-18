package search

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/elmerred09/blog/internal/validation"
)

func TestQueryValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		q      Query
		fields []string
	}{
		{"word", Query{Text: "postgres", Limit: DefaultLimit}, nil},
		{"phrase with operators", Query{Text: `"keyset pagination" -offset`, Limit: 1}, nil},
		{"max limit", Query{Text: "go", Limit: MaxLimit}, nil},
		{"max length", Query{Text: strings.Repeat("a", MaxQueryLen), Limit: 1}, nil},
		{"max length in runes, not bytes", Query{Text: strings.Repeat("é", MaxQueryLen), Limit: 1}, nil},
		{"surrounding space doesn't count", Query{Text: "  " + strings.Repeat("a", MaxQueryLen) + "\n", Limit: 1}, nil},
		{"empty", Query{Text: "", Limit: 1}, []string{"q"}},
		{"blank", Query{Text: " \t\n", Limit: 1}, []string{"q"}},
		{"too long", Query{Text: strings.Repeat("a", MaxQueryLen+1), Limit: 1}, []string{"q"}},
		{"limit zero", Query{Text: "go", Limit: 0}, []string{"limit"}},
		{"limit negative", Query{Text: "go", Limit: -1}, []string{"limit"}},
		{"limit too big", Query{Text: "go", Limit: MaxLimit + 1}, []string{"limit"}},
		{"both", Query{Text: "", Limit: 0}, []string{"q", "limit"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.q.Validate()
			if tc.fields == nil {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			var invalid *validation.Error
			if !errors.As(err, &invalid) {
				t.Fatalf("Validate() = %v, want a *validation.Error", err)
			}
			got := make([]string, len(invalid.Problems))
			for i, p := range invalid.Problems {
				got[i] = p.Field
				if p.Message == "" {
					t.Errorf("problem %s has an empty message", p.Field)
				}
			}
			if !slices.Equal(got, tc.fields) {
				t.Errorf("fields = %v, want %v", got, tc.fields)
			}
		})
	}
}

func TestSplitSnippet(t *testing.T) {
	t.Parallel()

	const s, e = matchStart, matchStop
	tests := []struct {
		name string
		in   string
		want []Fragment
	}{
		{"empty", "", []Fragment{}},
		{"no match", "plain text", []Fragment{{"plain text", false}}},
		{"middle", "use " + s + "keyset" + e + " pagination", []Fragment{
			{"use ", false}, {"keyset", true}, {" pagination", false},
		}},
		{"leading and trailing", s + "a" + e + " b " + s + "c" + e, []Fragment{
			{"a", true}, {" b ", false}, {"c", true},
		}},
		{"adjacent matches merge", s + "a" + e + s + "b" + e, []Fragment{{"ab", true}}},
		{"empty match is dropped", "a" + s + e + "b", []Fragment{{"ab", false}}},
		{"stray stop is dropped", "a" + e + "b", []Fragment{{"ab", false}}},
		{"repeated start is dropped", s + "a" + s + "b" + e, []Fragment{{"ab", true}}},
		{"unclosed match runs to the end", "a " + s + "b", []Fragment{{"a ", false}, {"b", true}}},
		{"HTML stays text", "<script>" + s + "x" + e + "</script> &amp;", []Fragment{
			{"<script>", false}, {"x", true}, {"</script> &amp;", false},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := splitSnippet(tc.in)
			if got == nil {
				t.Fatal("splitSnippet returned nil, want a non-nil slice")
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("splitSnippet(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}
