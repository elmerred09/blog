package content

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type frontmatter struct {
	Title       string     `yaml:"title"`
	Summary     string     `yaml:"summary"`
	Slug        string     `yaml:"slug"`
	PublishedAt *time.Time `yaml:"published_at"`
	Tags        []string   `yaml:"tags"`
}

// Parse reads one markdown file: YAML frontmatter between --- lines, then the
// body. name is the file's slash-separated path within the content FS; its
// base name is the slug when the frontmatter has none. All validation
// problems are reported together.
func Parse(name string, src []byte) (Source, error) {
	rawMeta, body, err := splitFrontmatter(src)
	if err != nil {
		return Source{}, err
	}
	meta, err := decodeFrontmatter(rawMeta)
	if err != nil {
		return Source{}, err
	}
	return meta.toSource(name, body)
}

func splitFrontmatter(src []byte) (rawMeta, body []byte, err error) {
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))

	rest, ok := bytes.CutPrefix(src, []byte("---\n"))
	if !ok {
		return nil, nil, errors.New("missing frontmatter: file must start with a --- line")
	}

	// The leading newline lets an empty block (---\n---\n) match too.
	rawMeta, body, ok = bytes.Cut(append([]byte("\n"), rest...), []byte("\n---\n"))
	if !ok {
		return nil, nil, errors.New("unclosed frontmatter: no closing --- line")
	}
	return rawMeta, body, nil
}

func decodeFrontmatter(rawMeta []byte) (frontmatter, error) {
	var meta frontmatter
	dec := yaml.NewDecoder(bytes.NewReader(rawMeta))
	dec.KnownFields(true)

	// io.EOF means the block was empty; toSource reports the missing fields.
	if err := dec.Decode(&meta); err != nil && !errors.Is(err, io.EOF) {
		return frontmatter{}, fmt.Errorf("frontmatter: %w", err)
	}
	return meta, nil
}

func (meta frontmatter) toSource(name string, body []byte) (Source, error) {
	var errs []error

	if strings.TrimSpace(meta.Title) == "" {
		errs = append(errs, errors.New("title is required"))
	}
	if strings.TrimSpace(meta.Summary) == "" {
		errs = append(errs, errors.New("summary is required"))
	}

	slug, err := resolveSlug(name, meta.Slug)
	if err != nil {
		errs = append(errs, err)
	}

	tags, tagErrs := resolveTags(meta.Tags)
	errs = append(errs, tagErrs...)

	if len(errs) > 0 {
		return Source{}, errors.Join(errs...)
	}
	return Source{
		Path:        name,
		Slug:        slug,
		Title:       meta.Title,
		Summary:     meta.Summary,
		PublishedAt: meta.PublishedAt,
		Tags:        tags,
		BodyMD:      string(body),
	}, nil
}

// resolveSlug validates an explicit frontmatter slug as is, or derives one
// from the file name.
func resolveSlug(name, explicit string) (string, error) {
	if explicit != "" {
		if !ValidSlug(explicit) {
			return "", fmt.Errorf("slug %q must be lowercase letters, digits, and single dashes", explicit)
		}
		return explicit, nil
	}

	stem := strings.TrimSuffix(path.Base(name), path.Ext(name))
	slug, err := Slugify(stem)
	if err != nil {
		return "", fmt.Errorf("slug from file name: %w", err)
	}
	return slug, nil
}

// resolveTags slugifies tag names, keeping file order. Names that share a
// slug collapse to the first one.
func resolveTags(names []string) ([]Tag, []error) {
	var (
		tags []Tag
		errs []error
		seen = make(map[string]bool, len(names))
	)
	for _, name := range names {
		name = strings.TrimSpace(name)
		slug, err := Slugify(name)
		if err != nil {
			errs = append(errs, fmt.Errorf("tag: %w", err))
			continue
		}
		if seen[slug] {
			continue
		}
		seen[slug] = true
		tags = append(tags, Tag{Slug: slug, Name: name})
	}
	return tags, errs
}
