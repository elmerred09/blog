package content_test

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/content"
	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

func mdFile(frontmatter, body string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("---\n" + frontmatter + "---\n" + body)}
}

// contentFS is the starting content for most tests: one published post with
// tags and one untagged draft in a subdirectory.
func contentFS() fstest.MapFS {
	return fstest.MapFS{
		"keyset.md": mdFile(
			"title: Keyset pagination\nsummary: Why OFFSET gets slow.\npublished_at: 2026-10-01\ntags: [Postgres, Performance]\n",
			"# Keyset\n\n```go\nfunc main() {}\n```\n",
		),
		"drafts/idea.md": mdFile("title: An idea\nsummary: Not ready.\n", "Some text.\n"),
	}
}

type importerDB struct {
	pool *pgxpool.Pool
	im   *content.Importer
}

func newImporterDB(t *testing.T) importerDB {
	t.Helper()
	pool, _ := env.NewDB(t)
	return importerDB{
		pool: pool,
		im:   content.NewImporter(pool, content.NewRenderer()),
	}
}

func (d importerDB) mustImport(t *testing.T, fsys fstest.MapFS, paths ...string) content.Summary {
	t.Helper()
	s, err := d.im.Import(t.Context(), fsys, paths)
	if err != nil {
		t.Fatalf("Import(%q): %v", paths, err)
	}
	return s
}

type storedPost struct {
	ID          uuid.UUID
	Title       string
	BodyHTML    string
	PublishedAt *time.Time
	DeletedAt   *time.Time
	UpdatedAt   time.Time
	Xmin        string
}

func (d importerDB) post(t *testing.T, slug string) storedPost {
	t.Helper()
	var p storedPost
	err := d.pool.QueryRow(t.Context(), `
SELECT id, title, body_html, published_at, deleted_at, updated_at, xmin::text
FROM posts WHERE slug = $1`, slug,
	).Scan(&p.ID, &p.Title, &p.BodyHTML, &p.PublishedAt, &p.DeletedAt, &p.UpdatedAt, &p.Xmin)
	if err != nil {
		t.Fatalf("select post %q: %v", slug, err)
	}
	return p
}

// links returns the post's tag links as tag slug -> xmin of the post_tags row.
func (d importerDB) links(t *testing.T, postSlug string) map[string]string {
	t.Helper()
	rows, err := d.pool.Query(t.Context(), `
SELECT t.slug, pt.xmin::text
FROM post_tags pt
JOIN tags t ON t.id = pt.tag_id
JOIN posts p ON p.id = pt.post_id
WHERE p.slug = $1`, postSlug)
	if err != nil {
		t.Fatalf("select links for %q: %v", postSlug, err)
	}
	defer rows.Close()

	links := map[string]string{}
	for rows.Next() {
		var slug, xmin string
		if err := rows.Scan(&slug, &xmin); err != nil {
			t.Fatalf("scan link: %v", err)
		}
		links[slug] = xmin
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("links for %q: %v", postSlug, err)
	}
	return links
}

func (d importerDB) tagName(t *testing.T, slug string) string {
	t.Helper()
	var name string
	if err := d.pool.QueryRow(t.Context(), `SELECT name FROM tags WHERE slug = $1`, slug).Scan(&name); err != nil {
		t.Fatalf("select tag %q: %v", slug, err)
	}
	return name
}

func (d importerDB) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := d.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func outcomes(s content.Summary) map[string]content.Outcome {
	m := make(map[string]content.Outcome, len(s.Files))
	for _, f := range s.Files {
		m[f.Slug] = f.Outcome
	}
	return m
}

func linkSlugs(links map[string]string) []string {
	return slices.Sorted(maps.Keys(links))
}

func TestImportFirstRun(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)

	s := d.mustImport(t, contentFS(), ".")

	wantFiles := []content.FileResult{
		{Path: "drafts/idea.md", Slug: "idea", Outcome: content.Inserted},
		{Path: "keyset.md", Slug: "keyset", Outcome: content.Inserted},
	}
	if !slices.Equal(s.Files, wantFiles) {
		t.Errorf("Files = %+v, want %+v", s.Files, wantFiles)
	}
	if s.TagsCreated != 2 || s.TagLinksAdded != 2 || s.TagLinksRemoved != 0 {
		t.Errorf("tags created/added/removed = %d/%d/%d, want 2/2/0", s.TagsCreated, s.TagLinksAdded, s.TagLinksRemoved)
	}

	keyset := d.post(t, "keyset")
	if keyset.Title != "Keyset pagination" {
		t.Errorf("title = %q", keyset.Title)
	}
	if !strings.Contains(keyset.BodyHTML, `class="chroma"`) {
		t.Errorf("body_html was not rendered with the Renderer:\n%s", keyset.BodyHTML)
	}
	wantAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if keyset.PublishedAt == nil || !keyset.PublishedAt.Equal(wantAt) {
		t.Errorf("published_at = %v, want %v", keyset.PublishedAt, wantAt)
	}
	if got := linkSlugs(d.links(t, "keyset")); !slices.Equal(got, []string{"performance", "postgres"}) {
		t.Errorf("keyset tags = %q, want [performance postgres]", got)
	}
	if got := d.tagName(t, "postgres"); got != "Postgres" {
		t.Errorf("tag name = %q, want %q", got, "Postgres")
	}

	idea := d.post(t, "idea")
	if idea.PublishedAt != nil {
		t.Errorf("draft published_at = %v, want NULL", *idea.PublishedAt)
	}
	if got := d.links(t, "idea"); len(got) != 0 {
		t.Errorf("draft tags = %q, want none", linkSlugs(got))
	}
}

func TestImportReimportWritesNothing(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := contentFS()

	d.mustImport(t, fsys, ".")
	beforeKeyset, beforeIdea := d.post(t, "keyset"), d.post(t, "idea")
	beforeLinks := d.links(t, "keyset")

	s := d.mustImport(t, fsys, ".")

	if got := s.Count(content.Unchanged); got != 2 {
		t.Errorf("unchanged = %d, want 2 (files: %+v)", got, s.Files)
	}
	if s.TagsCreated != 0 || s.TagLinksAdded != 0 || s.TagLinksRemoved != 0 {
		t.Errorf("tags created/added/removed = %d/%d/%d, want 0/0/0", s.TagsCreated, s.TagLinksAdded, s.TagLinksRemoved)
	}
	for _, c := range []struct {
		slug          string
		before, after storedPost
	}{
		{"keyset", beforeKeyset, d.post(t, "keyset")},
		{"idea", beforeIdea, d.post(t, "idea")},
	} {
		if c.after.Xmin != c.before.Xmin {
			t.Errorf("%s: xmin %s -> %s, want the row left alone", c.slug, c.before.Xmin, c.after.Xmin)
		}
		if !c.after.UpdatedAt.Equal(c.before.UpdatedAt) {
			t.Errorf("%s: updated_at %v -> %v, want unchanged", c.slug, c.before.UpdatedAt, c.after.UpdatedAt)
		}
	}
	if afterLinks := d.links(t, "keyset"); !maps.Equal(afterLinks, beforeLinks) {
		t.Errorf("post_tags (slug -> xmin) %v -> %v, want untouched", beforeLinks, afterLinks)
	}
}

func TestImportEditedFile(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := contentFS()

	d.mustImport(t, fsys, ".")
	beforeKeyset, beforeIdea := d.post(t, "keyset"), d.post(t, "idea")

	fsys["keyset.md"] = mdFile(
		"title: Keyset pagination\nsummary: Why OFFSET gets slow.\npublished_at: 2026-10-01\ntags: [Postgres, Performance]\n",
		"# Keyset\n\nNow with a second paragraph.\n",
	)
	s := d.mustImport(t, fsys, ".")

	want := map[string]content.Outcome{"keyset": content.Updated, "idea": content.Unchanged}
	if got := outcomes(s); !maps.Equal(got, want) {
		t.Errorf("outcomes = %v, want %v", got, want)
	}

	keyset := d.post(t, "keyset")
	if keyset.ID != beforeKeyset.ID {
		t.Errorf("id changed %s -> %s; an update must keep the id", beforeKeyset.ID, keyset.ID)
	}
	if !strings.Contains(keyset.BodyHTML, "second paragraph") {
		t.Errorf("body_html not re-rendered:\n%s", keyset.BodyHTML)
	}
	if keyset.Xmin == beforeKeyset.Xmin {
		t.Error("keyset xmin unchanged; the edited post should have been rewritten")
	}
	if idea := d.post(t, "idea"); idea.Xmin != beforeIdea.Xmin {
		t.Errorf("idea xmin %s -> %s; an unedited file must not be rewritten", beforeIdea.Xmin, idea.Xmin)
	}
}

func TestImportPublishingADraft(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := contentFS()

	d.mustImport(t, fsys, ".")
	fsys["drafts/idea.md"] = mdFile("title: An idea\nsummary: Not ready.\npublished_at: 2026-10-05\n", "Some text.\n")
	s := d.mustImport(t, fsys, "drafts/idea.md")

	if got := outcomes(s); got["idea"] != content.Updated {
		t.Errorf("idea outcome = %q, want %q (NULL -> a date is a change)", got["idea"], content.Updated)
	}
	if idea := d.post(t, "idea"); idea.PublishedAt == nil {
		t.Error("published_at still NULL after publishing")
	}
}

func TestImportTagChanges(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := fstest.MapFS{
		"a.md": mdFile("title: A\nsummary: S\ntags: [Postgres, Go, SQL]\n", "body\n"),
	}

	d.mustImport(t, fsys, "a.md")
	before := d.links(t, "a")

	fsys["a.md"] = mdFile("title: A\nsummary: S\ntags: [Postgres, Search]\n", "body\n")
	s := d.mustImport(t, fsys, "a.md")

	if got := outcomes(s)["a"]; got != content.Unchanged {
		t.Errorf("outcome = %q, want %q: only tags changed, the posts row didn't", got, content.Unchanged)
	}
	if s.TagsCreated != 1 || s.TagLinksAdded != 1 || s.TagLinksRemoved != 2 {
		t.Errorf("tags created/added/removed = %d/%d/%d, want 1/1/2", s.TagsCreated, s.TagLinksAdded, s.TagLinksRemoved)
	}

	after := d.links(t, "a")
	if got := linkSlugs(after); !slices.Equal(got, []string{"postgres", "search"}) {
		t.Errorf("tags = %q, want [postgres search]", got)
	}
	if after["postgres"] != before["postgres"] {
		t.Errorf("postgres link xmin %s -> %s; a link that stays must not be rewritten", before["postgres"], after["postgres"])
	}
	if got := d.count(t, "tags"); got != 4 {
		t.Errorf("tags table has %d rows, want 4: unused tags are not deleted", got)
	}
}

func TestImportSubsetLeavesOtherPostsAlone(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := contentFS()

	handMade := dbtest.InsertPost(t, d.pool, dbtest.WithSlug("hand-made"))
	d.mustImport(t, fsys, ".")
	beforeKeyset := d.post(t, "keyset")

	// keyset.md changes on disk, but only the draft is named.
	fsys["keyset.md"] = mdFile("title: Retitled\nsummary: S\n", "body\n")
	s := d.mustImport(t, fsys, "drafts/idea.md")

	wantFiles := []content.FileResult{{Path: "drafts/idea.md", Slug: "idea", Outcome: content.Unchanged}}
	if !slices.Equal(s.Files, wantFiles) {
		t.Errorf("Files = %+v, want %+v", s.Files, wantFiles)
	}
	if keyset := d.post(t, "keyset"); keyset.Xmin != beforeKeyset.Xmin || keyset.Title != "Keyset pagination" {
		t.Errorf("keyset was rewritten (title %q) although its file wasn't named", keyset.Title)
	}
	if got := d.post(t, "hand-made"); got.ID != handMade.ID || got.DeletedAt != nil {
		t.Errorf("post with no file was changed: %+v", got)
	}
}

func TestImportRestoresSoftDeletedPost(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := contentFS()

	d.mustImport(t, fsys, ".")
	before := d.post(t, "keyset")
	if _, err := d.pool.Exec(t.Context(), `UPDATE posts SET deleted_at = now() WHERE slug = 'keyset'`); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	s := d.mustImport(t, fsys, "keyset.md")

	if got := outcomes(s)["keyset"]; got != content.Updated {
		t.Errorf("outcome = %q, want %q (restoring is a write)", got, content.Updated)
	}
	after := d.post(t, "keyset")
	if after.DeletedAt != nil {
		t.Errorf("deleted_at = %v, want NULL after re-import", *after.DeletedAt)
	}
	if after.ID != before.ID {
		t.Errorf("id changed %s -> %s; restoring must keep the id", before.ID, after.ID)
	}
}

func TestImportTagNameComesFromFirstFile(t *testing.T) {
	t.Parallel()
	d := newImporterDB(t)
	fsys := fstest.MapFS{
		"a.md": mdFile("title: A\nsummary: S\ntags: [Postgres]\n", "body\n"),
		"b.md": mdFile("title: B\nsummary: S\ntags: [POSTGRES]\n", "body\n"),
	}

	s := d.mustImport(t, fsys, ".")
	if s.TagsCreated != 1 {
		t.Errorf("tags created = %d, want 1 (same slug)", s.TagsCreated)
	}
	if got := d.tagName(t, "postgres"); got != "Postgres" {
		t.Errorf("tag name = %q, want %q from a.md, the first file in path order", got, "Postgres")
	}

	fsys["b.md"] = mdFile("title: B\nsummary: S\ntags: [PostGres]\n", "body\n")
	d.mustImport(t, fsys, "b.md")
	if got := d.tagName(t, "postgres"); got != "Postgres" {
		t.Errorf("tag name = %q after a later import, want %q: existing tags keep their name", got, "Postgres")
	}
}

func TestImportErrorsWriteNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fsys     fstest.MapFS
		wantErrs []string
	}{
		{
			name: "parse error in one file",
			fsys: fstest.MapFS{
				"good.md": mdFile("title: Good\nsummary: S\ntags: [Go]\n", "body\n"),
				"bad.md":  mdFile("summary: no title\n", "body\n"),
			},
			wantErrs: []string{"bad.md", "title"},
		},
		{
			name: "duplicate slugs across files",
			fsys: fstest.MapFS{
				"a.md": mdFile("title: A\nsummary: S\nslug: same\n", "body\n"),
				"b.md": mdFile("title: B\nsummary: S\nslug: same\n", "body\n"),
			},
			wantErrs: []string{"same", "a.md", "b.md"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := newImporterDB(t)

			s, err := d.im.Import(t.Context(), tc.fsys, []string{"."})
			if err == nil {
				t.Fatalf("Import succeeded with %+v, want an error", s)
			}
			for _, want := range tc.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
			for _, table := range []string{"posts", "tags", "post_tags"} {
				if n := d.count(t, table); n != 0 {
					t.Errorf("%s has %d rows after a failed import, want 0", table, n)
				}
			}
		})
	}
}
