package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/db"
)

const rerenderBatch = 100

// RerenderResult counts the posts Rerender read and the ones whose stored
// HTML it replaced.
type RerenderResult struct {
	Checked int
	Updated int
}

// Rerender renders every non-deleted post's markdown again and stores the HTML
// where it differs, so a renderer change reaches posts saved before it. A post
// saved while this runs keeps the HTML that save stored. updated_at is not
// touched, so an editor open on a post can still save it.
func (s *Service) Rerender(ctx context.Context) (RerenderResult, error) {
	var res RerenderResult
	after := uuid.Nil
	for {
		rows, err := s.q.ListPostBodies(ctx, db.ListPostBodiesParams{AfterID: after, BatchSize: rerenderBatch})
		if err != nil {
			return res, fmt.Errorf("list post bodies after %s: %w", after, err)
		}
		for _, row := range rows {
			res.Checked++
			html, err := s.render.Render([]byte(row.BodyMd))
			if err != nil {
				return res, fmt.Errorf("render post %s: %w", row.ID, err)
			}
			if html == row.BodyHtml {
				continue
			}
			n, err := s.q.SetPostHTML(ctx, db.SetPostHTMLParams{ID: row.ID, BodyMd: row.BodyMd, BodyHtml: html})
			if err != nil {
				return res, fmt.Errorf("store html for post %s: %w", row.ID, err)
			}
			res.Updated += int(n)
		}
		if len(rows) < rerenderBatch {
			return res, nil
		}
		after = rows[len(rows)-1].ID
	}
}
