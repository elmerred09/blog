package content

import (
	"time"

	"github.com/elmerred09/blog/internal/tags"
)

type Source struct {
	Path        string
	Slug        string
	Title       string
	Summary     string
	PublishedAt *time.Time
	Tags        []tags.Tag
	BodyMD      string
}
