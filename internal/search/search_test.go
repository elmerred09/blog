package search

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestQueryValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		q       Query
		wantErr bool
	}{
		{"word", Query{Text: "postgres", Limit: DefaultLimit}, false},
		{"phrase with operators", Query{Text: `"keyset pagination" -offset`, Limit: 1}, false},
		{"max limit", Query{Text: "go", Limit: MaxLimit}, false},
		{"max length", Query{Text: strings.Repeat("a", MaxQueryLen), Limit: 1}, false},
		{"max length in runes, not bytes", Query{Text: strings.Repeat("é", MaxQueryLen), Limit: 1}, false},
		{"surrounding space doesn't count", Query{Text: "  " + strings.Repeat("a", MaxQueryLen) + "\n", Limit: 1}, false},
		{"empty", Query{Text: "", Limit: 1}, true},
		{"blank", Query{Text: " \t\n", Limit: 1}, true},
		{"too long", Query{Text: strings.Repeat("a", MaxQueryLen+1), Limit: 1}, true},
		{"limit zero", Query{Text: "go", Limit: 0}, true},
		{"limit negative", Query{Text: "go", Limit: -1}, true},
		{"limit too big", Query{Text: "go", Limit: MaxLimit + 1}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.q.Validate()
			if !tc.wantErr {
				if err != nil {
					t.Errorf("Validate() = %v, want nil", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Message == "" {
				t.Errorf("Validate() = %v, want a *ValidationError with a message", err)
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
