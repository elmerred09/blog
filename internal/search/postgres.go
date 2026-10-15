package search

import (
	"context"
	"fmt"

	"github.com/elmerred09/blog/internal/db"
	"github.com/elmerred09/blog/internal/post"
)

// Postgres searches posts.search_vector with Postgres full-text search.
type Postgres struct {
	q *db.Queries
}

var _ Searcher = (*Postgres)(nil)

func NewPostgres(conn db.DBTX) *Postgres {
	return &Postgres{q: db.New(conn)}
}

func (p *Postgres) Search(ctx context.Context, q Query) ([]Result, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	rows, err := p.q.SearchPosts(ctx, db.SearchPostsParams{Query: q.Text, ResultLimit: int32(q.Limit)})
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}

	itemRows := make([]post.Row, len(rows))
	for i, r := range rows {
		itemRows[i] = post.Row{ID: r.ID, Slug: r.Slug, Title: r.Title, Summary: r.Summary, PublishedAt: r.PublishedAt}
	}
	items, err := post.ListItems(ctx, p.q, itemRows)
	if err != nil {
		return nil, fmt.Errorf("search posts: %w", err)
	}

	results := make([]Result, len(rows))
	for i, r := range rows {
		results[i] = Result{ListItem: items[i], Snippet: splitSnippet(r.Snippet)}
	}
	return results, nil
}
