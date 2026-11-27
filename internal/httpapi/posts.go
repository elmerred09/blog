package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/post"
)

type tagJSON struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type listItemJSON struct {
	Slug           string    `json:"slug"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	PublishedAt    time.Time `json:"published_at"`
	ReadingMinutes int       `json:"reading_minutes"`
	Tags           []tagJSON `json:"tags"`
}

type pageJSON struct {
	Items      []listItemJSON `json:"items"`
	NextCursor *string        `json:"next_cursor"`
	PrevCursor *string        `json:"prev_cursor"`
}

type postJSON struct {
	listItemJSON
	BodyHTML  string    `json:"body_html"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toListItemJSON(it post.ListItem) listItemJSON {
	tags := make([]tagJSON, len(it.Tags))
	for i, t := range it.Tags {
		tags[i] = tagJSON(t)
	}
	return listItemJSON{
		Slug:           it.Slug,
		Title:          it.Title,
		Summary:        it.Summary,
		PublishedAt:    it.PublishedAt.UTC(),
		ReadingMinutes: it.ReadingMinutes,
		Tags:           tags,
	}
}

var invalidLimitMessage = fmt.Sprintf("limit must be an integer from 1 to %d", post.MaxLimit)

// listParams reads ?limit= and ?cursor=. On a bad value it writes the 400 and
// returns false.
func listParams(c *gin.Context) (post.ListParams, bool) {
	params := post.ListParams{Limit: post.DefaultLimit}
	if raw, ok := c.GetQuery("limit"); ok {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, invalidLimitMessage)
			return post.ListParams{}, false
		}
		params.Limit = n
	}
	if raw, ok := c.GetQuery("cursor"); ok {
		cursor, err := post.ParseCursor(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, "invalid cursor")
			return post.ListParams{}, false
		}
		params.Cursor = &cursor
	}
	return params, true
}

func toPageJSON(page post.Page) pageJSON {
	resp := pageJSON{Items: make([]listItemJSON, len(page.Items))}
	for i, it := range page.Items {
		resp.Items[i] = toListItemJSON(it)
	}
	if page.Next != nil {
		next := page.Next.String()
		resp.NextCursor = &next
	}
	if page.Prev != nil {
		prev := page.Prev.String()
		resp.PrevCursor = &prev
	}
	return resp
}

func listPosts(posts *post.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		params, ok := listParams(c)
		if !ok {
			return
		}
		page, err := posts.List(c.Request.Context(), params)
		if errors.Is(err, post.ErrInvalidLimit) {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, invalidLimitMessage)
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, toPageJSON(page))
	}
}

type tagPageJSON struct {
	Tag tagJSON `json:"tag"`
	pageJSON
}

func listPostsByTag(posts *post.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		params, ok := listParams(c)
		if !ok {
			return
		}
		tp, err := posts.ListByTag(c.Request.Context(), c.Param("slug"), params)
		if errors.Is(err, post.ErrInvalidLimit) {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, invalidLimitMessage)
			return
		}
		if errors.Is(err, post.ErrNotFound) {
			writeError(c, http.StatusNotFound, codeNotFound, "no published posts with this tag")
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, tagPageJSON{Tag: tagJSON(tp.Tag), pageJSON: toPageJSON(tp.Page)})
	}
}

type sitemapItemJSON struct {
	Slug         string    `json:"slug"`
	LastModified time.Time `json:"last_modified"`
}

type sitemapJSON struct {
	Items []sitemapItemJSON `json:"items"`
}

func getSitemap(posts *post.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := posts.SitemapEntries(c.Request.Context())
		if err != nil {
			internalError(c, err)
			return
		}
		resp := sitemapJSON{Items: make([]sitemapItemJSON, len(entries))}
		for i, e := range entries {
			resp.Items[i] = sitemapItemJSON{Slug: e.Slug, LastModified: e.LastModified.UTC()}
		}
		c.JSON(http.StatusOK, resp)
	}
}

type neighborJSON struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type adjacentJSON struct {
	Newer *neighborJSON `json:"newer"`
	Older *neighborJSON `json:"older"`
}

func toNeighborJSON(n *post.Neighbor) *neighborJSON {
	if n == nil {
		return nil
	}
	return &neighborJSON{Slug: n.Slug, Title: n.Title}
}

func getAdjacentPosts(posts *post.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		adj, err := posts.Adjacent(c.Request.Context(), c.Param("slug"))
		if errors.Is(err, post.ErrNotFound) {
			writeError(c, http.StatusNotFound, codeNotFound, "post not found")
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, adjacentJSON{Newer: toNeighborJSON(adj.Newer), Older: toNeighborJSON(adj.Older)})
	}
}

func getPost(posts *post.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := posts.Get(c.Request.Context(), c.Param("slug"))
		if errors.Is(err, post.ErrNotFound) {
			writeError(c, http.StatusNotFound, codeNotFound, "post not found")
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}

		c.JSON(http.StatusOK, postJSON{
			listItemJSON: toListItemJSON(p.ListItem),
			BodyHTML:     p.BodyHTML,
			UpdatedAt:    p.UpdatedAt.UTC(),
		})
	}
}
