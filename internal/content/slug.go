package content

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Slugify turns a display name into a slug: lowercase ASCII letters and
// digits, with each run of anything else collapsed to a single dash and none
// at either end. Accents are folded ("Café" becomes "cafe"). It returns an
// error if nothing usable is left.
func Slugify(name string) (string, error) {
	var slug strings.Builder
	pendingDash := false

	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			// An accent split off by NFD; dropping it keeps the base letter.
		case isSlugChar(r):
			if pendingDash && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			pendingDash = false
			slug.WriteRune(r)
		default:
			pendingDash = true
		}
	}

	if slug.Len() == 0 {
		return "", fmt.Errorf("slugify %q: no letters or digits", name)
	}
	return slug.String(), nil
}

// ValidSlug reports whether s is already a well-formed slug.
func ValidSlug(s string) bool {
	return slugPattern.MatchString(s)
}

// isSlugChar reports whether r can appear in a slug as is. The input is
// already lowercased, so uppercase letters don't need a case here.
func isSlugChar(r rune) bool {
	return ('a' <= r && r <= 'z') || ('0' <= r && r <= '9')
}
