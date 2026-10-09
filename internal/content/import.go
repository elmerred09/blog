package content

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"

	"github.com/elmerred09/blog/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Importer struct {
	pool   *pgxpool.Pool
	render *Renderer
}

// Outcome is what an import did to a post's row. Tag link changes are counted
// separately in Summary, so a post whose only change is its tags is unchanged.
type Outcome string

const (
	Inserted  Outcome = "inserted"
	Updated   Outcome = "updated"
	Unchanged Outcome = "unchanged"
)

type FileResult struct {
	Path    string
	Slug    string
	Outcome Outcome
}

type Summary struct {
	Files           []FileResult // in the order ResolvePaths returned them
	TagsCreated     int
	TagLinksAdded   int
	TagLinksRemoved int
}

// Count returns how many files had outcome o.
func (s Summary) Count(o Outcome) int {
	n := 0
	for _, f := range s.Files {
		if f.Outcome == o {
			n++
		}
	}
	return n
}

func NewImporter(pool *pgxpool.Pool, render *Renderer) *Importer {
	return &Importer{pool: pool, render: render}
}

// Import upserts the markdown files named by paths (files or directories
// within fsys). Every file is parsed before the database is touched, and all
// writes happen in one transaction. Nothing is ever deleted.
func (im *Importer) Import(ctx context.Context, fsys fs.FS, paths []string) (Summary, error) {
	names, err := ResolvePaths(fsys, paths)
	if err != nil {
		return Summary{}, fmt.Errorf("resolving paths: %w", err)
	}
	sources, err := loadAll(fsys, names)
	if err != nil {
		return Summary{}, fmt.Errorf("loading files: %w", err)
	}
	html, err := im.renderAll(sources)
	if err != nil {
		return Summary{}, fmt.Errorf("rendering posts: %w", err)
	}

	tx, err := im.pool.Begin(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)

	var summary Summary
	var postIDs map[string]uuid.UUID
	summary.Files, postIDs, err = upsertPosts(ctx, q, sources, html)
	if err != nil {
		return Summary{}, err
	}
	tagIDs, created, err := upsertTags(ctx, q, sources)
	if err != nil {
		return Summary{}, err
	}
	summary.TagsCreated = created
	summary.TagLinksAdded, summary.TagLinksRemoved, err = syncLinks(ctx, q, sources, postIDs, tagIDs)
	if err != nil {
		return Summary{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Summary{}, fmt.Errorf("commit: %w", err)
	}
	return summary, nil
}

// upsertPosts writes each source's post row (html[i] belongs to sources[i])
// and returns one result per file plus the post ids by slug.
func upsertPosts(ctx context.Context, q *db.Queries, sources []Source, html []string) ([]FileResult, map[string]uuid.UUID, error) {
	files := make([]FileResult, 0, len(sources))
	postIDs := make(map[string]uuid.UUID, len(sources))

	for i, source := range sources {
		row, err := q.UpsertPost(ctx, db.UpsertPostParams{
			Slug:        source.Slug,
			Title:       source.Title,
			Summary:     source.Summary,
			BodyMd:      source.BodyMD,
			BodyHtml:    html[i],
			PublishedAt: source.PublishedAt,
		})

		var outcome Outcome
		switch {
		case errors.Is(err, pgx.ErrNoRows): // content unchanged, so the upsert skipped the row
			row.ID, err = q.GetPostIDBySlug(ctx, source.Slug)
			if err != nil {
				return nil, nil, fmt.Errorf("get post id %q: %w", source.Slug, err)
			}
			outcome = Unchanged
		case err != nil:
			return nil, nil, fmt.Errorf("upsert post %q: %w", source.Slug, err)
		case row.Created:
			outcome = Inserted
		default:
			outcome = Updated
		}

		files = append(files, FileResult{Path: source.Path, Slug: source.Slug, Outcome: outcome})
		postIDs[source.Slug] = row.ID
	}
	return files, postIDs, nil
}

// upsertTags creates any missing tags used by sources and returns every tag
// id by slug, plus how many tags were created. A new tag is named by the
// first source (in path order) that uses it; existing tags keep their name.
func upsertTags(ctx context.Context, q *db.Queries, sources []Source) (map[string]int32, int, error) {
	tagsBySlug := make(map[string]Tag)
	for _, source := range sources {
		for _, tag := range source.Tags {
			if _, ok := tagsBySlug[tag.Slug]; !ok {
				tagsBySlug[tag.Slug] = tag
			}
		}
	}

	slugs := slices.Sorted(maps.Keys(tagsBySlug))
	created := 0
	for _, slug := range slugs {
		n, err := q.UpsertTag(ctx, db.UpsertTagParams{Slug: slug, Name: tagsBySlug[slug].Name})
		if err != nil {
			return nil, 0, fmt.Errorf("upsert tag %q: %w", slug, err)
		}
		created += int(n)
	}

	rows, err := q.GetTagsBySlugs(ctx, slugs)
	if err != nil {
		return nil, 0, fmt.Errorf("get tags by slugs: %w", err)
	}
	tagIDs := make(map[string]int32, len(rows))
	for _, row := range rows {
		tagIDs[row.Slug] = row.ID
	}
	return tagIDs, created, nil
}

// syncLinks makes each post's tag links match its source, touching only the
// links that differ. It returns how many links were added and removed.
func syncLinks(ctx context.Context, q *db.Queries, sources []Source, postIDs map[string]uuid.UUID, tagIDs map[string]int32) (int, int, error) {
	var added, removed int
	for _, source := range sources {
		postID := postIDs[source.Slug]

		ids := make([]int32, 0, len(source.Tags))
		for _, tag := range source.Tags {
			id, ok := tagIDs[tag.Slug]
			if !ok {
				return 0, 0, fmt.Errorf("post %q: tag %q not found after upsert", source.Slug, tag.Slug)
			}
			ids = append(ids, id)
		}

		n, err := q.DeleteTagsForPostExcept(ctx, db.DeleteTagsForPostExceptParams{PostID: postID, TagIds: ids})
		if err != nil {
			return 0, 0, fmt.Errorf("delete stale tags for post %q: %w", source.Slug, err)
		}
		removed += int(n)

		n, err = q.SetTagsForPost(ctx, db.SetTagsForPostParams{PostID: postID, TagIds: ids})
		if err != nil {
			return 0, 0, fmt.Errorf("add tags for post %q: %w", source.Slug, err)
		}
		added += int(n)
	}
	return added, removed, nil
}

// ResolvePaths expands paths (markdown files or directories within fsys) into
// a sorted, de-duplicated list of markdown files. Directories are walked
// recursively and their non-markdown files skipped. It is an error to pass no
// paths, a path that doesn't exist or isn't fs.ValidPath, a non-markdown file
// by name, or a directory with no markdown files.
func ResolvePaths(fsys fs.FS, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths provided")
	}

	seen := make(map[string]bool)
	for _, p := range paths {
		if !fs.ValidPath(p) {
			return nil, fmt.Errorf("invalid path %q: must be relative, clean, and slash-separated", p)
		}

		info, err := fs.Stat(fsys, p)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("missing path %s", p)
			}
			return nil, fmt.Errorf("stat %s: %w", p, err)
		}

		if !info.IsDir() {
			if !isMarkdown(p) {
				return nil, fmt.Errorf("%s is not a markdown file", p)
			}
			seen[p] = true
			continue
		}

		found, err := markdownFiles(fsys, p)
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return nil, fmt.Errorf("directory %s has no markdown files", p)
		}
		for _, name := range found {
			seen[name] = true
		}
	}

	return slices.Sorted(maps.Keys(seen)), nil
}

// markdownFiles returns every markdown file under dir. fs.WalkDir already
// descends into subdirectories, so the callback only has to pick files.
func markdownFiles(fsys fs.FS, dir string) ([]string, error) {
	var names []string
	err := fs.WalkDir(fsys, dir, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isMarkdown(name) {
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", dir, err)
	}
	return names, nil
}

func isMarkdown(name string) bool {
	return path.Ext(name) == ".md"
}

// loadAll reads and parses every named file, reporting all problems together,
// including two files that resolve to the same slug.
func loadAll(fsys fs.FS, names []string) ([]Source, error) {
	sources := make([]Source, 0, len(names))
	var errs []error

	for _, name := range names {
		src, err := fs.ReadFile(fsys, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("read file %s: %w", name, err))
			continue
		}

		source, err := Parse(name, src)
		if err != nil {
			errs = append(errs, fmt.Errorf("parse %s: %w", name, err))
			continue
		}

		sources = append(sources, source)
	}

	seen := make(map[string]string, len(sources))
	for _, source := range sources {
		if prev, ok := seen[source.Slug]; ok {
			errs = append(errs, fmt.Errorf("duplicate slug %q in %s and %s", source.Slug, prev, source.Path))
			continue
		}
		seen[source.Slug] = source.Path
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return sources, nil
}

// renderAll returns the HTML for each source, in the same order as sources.
func (im *Importer) renderAll(sources []Source) ([]string, error) {
	html := make([]string, len(sources))
	var errs []error

	for i, source := range sources {
		h, err := im.render.Render([]byte(source.BodyMD))
		if err != nil {
			errs = append(errs, fmt.Errorf("render %s: %w", source.Path, err))
			continue
		}
		html[i] = h
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return html, nil
}
