package content

import "time"

type Source struct {
	Path        string
	Slug        string
	Title       string
	Summary     string
	PublishedAt *time.Time
	Tags        []Tag
	BodyMD      string
}

type Tag struct{ Slug, Name string }
