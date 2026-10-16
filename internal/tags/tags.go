package tags

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/slug"
)

type Tag struct{ Slug, Name string }

// FromNames turns display names into tags, keeping their order. Names that
// share a slug collapse to the first one. Every name with nothing
// slug-worthy in it is reported.
func FromNames(names []string) ([]Tag, error) {
	var (
		tags []Tag
		errs []error
		seen = make(map[string]bool, len(names))
	)
	for _, name := range names {
		name = strings.TrimSpace(name)
		s, err := slug.Make(name)
		if err != nil {
			errs = append(errs, fmt.Errorf("tag: %w", err))
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		tags = append(tags, Tag{Slug: s, Name: name})
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return tags, nil
}

// Ensure creates any missing tags and returns every tag's id by slug, plus
// how many were created. A new tag is named by the first entry with its slug;
// existing tags keep their name. Tags are upserted in slug order, so
// concurrent callers lock them in the same order.
func Ensure(ctx context.Context, q *db.Queries, tags []Tag) (map[string]int32, int, error) {
	names := make(map[string]string, len(tags))
	for _, t := range tags {
		if _, ok := names[t.Slug]; !ok {
			names[t.Slug] = t.Name
		}
	}
	if len(names) == 0 {
		return map[string]int32{}, 0, nil
	}

	slugs := slices.Sorted(maps.Keys(names))
	created := 0
	for _, s := range slugs {
		n, err := q.UpsertTag(ctx, db.UpsertTagParams{Slug: s, Name: names[s]})
		if err != nil {
			return nil, 0, fmt.Errorf("upsert tag %q: %w", s, err)
		}
		created += int(n)
	}

	rows, err := q.GetTagsBySlugs(ctx, slugs)
	if err != nil {
		return nil, 0, fmt.Errorf("get tags by slugs: %w", err)
	}
	ids := make(map[string]int32, len(rows))
	for _, r := range rows {
		ids[r.Slug] = r.ID
	}
	return ids, created, nil
}

// Link makes the post's tag links exactly tagIDs, touching only the links
// that differ, and returns how many it added and removed. No tags (nil or
// empty) removes every link.
func Link(ctx context.Context, q *db.Queries, postID uuid.UUID, tagIDs []int32) (added, removed int, err error) {
	n, err := q.DeleteTagsForPostExcept(ctx, db.DeleteTagsForPostExceptParams{PostID: postID, TagIds: tagIDs})
	if err != nil {
		return 0, 0, fmt.Errorf("delete stale tag links: %w", err)
	}
	removed = int(n)

	n, err = q.SetTagsForPost(ctx, db.SetTagsForPostParams{PostID: postID, TagIds: tagIDs})
	if err != nil {
		return 0, 0, fmt.Errorf("add tag links: %w", err)
	}
	return int(n), removed, nil
}
