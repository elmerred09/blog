package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/media"
)

var invalidAdminLimit = fmt.Sprintf("limit must be an integer from 1 to %d", admin.MaxLimit)

type adminPostJSON struct {
	ID          uuid.UUID   `json:"id"`
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	Summary     string      `json:"summary"`
	BodyMD      string      `json:"body_md"`
	PublishedAt *time.Time  `json:"published_at"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	State       admin.State `json:"state"`
	Tags        []tagJSON   `json:"tags"`
}

type adminListItemJSON struct {
	ID          uuid.UUID   `json:"id"`
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	PublishedAt *time.Time  `json:"published_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	State       admin.State `json:"state"`
}

type adminPageJSON struct {
	Items      []adminListItemJSON `json:"items"`
	NextCursor *string             `json:"next_cursor"`
}

type adminWriteJSON struct {
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Summary     string     `json:"summary"`
	BodyMD      string     `json:"body_md"`
	PublishedAt *time.Time `json:"published_at"`
	Tags        []string   `json:"tags"`
	UpdatedAt   string     `json:"updated_at"`
}

type previewJSON struct {
	BodyMD string `json:"body_md"`
}

type previewResultJSON struct {
	BodyHTML string `json:"body_html"`
}

type uploadJSON struct {
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type uploadResultJSON struct {
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	PublicURL string            `json:"public_url"`
}

func listAdminPosts(posts *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		params := admin.ListParams{Limit: admin.DefaultLimit}
		if raw, ok := c.GetQuery("limit"); ok {
			n, err := strconv.Atoi(raw)
			if err != nil {
				writeError(c, http.StatusBadRequest, codeInvalidRequest, invalidAdminLimit)
				return
			}
			params.Limit = n
		}
		if raw, ok := c.GetQuery("cursor"); ok {
			id, err := uuid.Parse(raw)
			if err != nil {
				writeError(c, http.StatusBadRequest, codeInvalidRequest, "invalid cursor")
				return
			}
			params.After = &id
		}

		page, err := posts.List(c.Request.Context(), params)
		if errors.Is(err, admin.ErrInvalidLimit) {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, invalidAdminLimit)
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}

		resp := adminPageJSON{Items: make([]adminListItemJSON, len(page.Items))}
		for i, it := range page.Items {
			resp.Items[i] = adminListItemJSON{
				ID:          it.ID,
				Slug:        it.Slug,
				Title:       it.Title,
				PublishedAt: utcPtr(it.PublishedAt),
				UpdatedAt:   it.UpdatedAt.UTC(),
				State:       it.State,
			}
		}
		if page.Next != nil {
			next := page.Next.String()
			resp.NextCursor = &next
		}
		c.JSON(http.StatusOK, resp)
	}
}

func createAdminPost(posts *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body adminWriteJSON
		if !decodeJSON(c, &body) {
			return
		}
		created, err := posts.Create(c.Request.Context(), body.input())
		if err != nil {
			writeAdminError(c, err)
			return
		}
		c.JSON(http.StatusCreated, toAdminPost(created))
	}
}

func getAdminPost(posts *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := postID(c)
		if !ok {
			return
		}
		p, err := posts.Get(c.Request.Context(), id)
		if err != nil {
			writeAdminError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAdminPost(p))
	}
}

func updateAdminPost(posts *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := postID(c)
		if !ok {
			return
		}
		var body adminWriteJSON
		if !decodeJSON(c, &body) {
			return
		}
		if body.UpdatedAt == "" {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, "updated_at is required")
			return
		}
		expected, err := time.Parse(time.RFC3339Nano, body.UpdatedAt)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeInvalidRequest, "updated_at is not a timestamp")
			return
		}
		saved, err := posts.Update(c.Request.Context(), admin.Update{
			ID:                id,
			ExpectedUpdatedAt: expected,
			Input:             body.input(),
		})
		if err != nil {
			writeAdminError(c, err)
			return
		}
		c.JSON(http.StatusOK, toAdminPost(saved))
	}
}

func previewPost(posts *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body previewJSON
		if !decodeJSON(c, &body) {
			return
		}
		html, err := posts.Preview(body.BodyMD)
		if err != nil {
			writeAdminError(c, err)
			return
		}
		c.JSON(http.StatusOK, previewResultJSON{BodyHTML: html})
	}
}

func createUpload(uploader *media.Uploader) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body uploadJSON
		if !decodeJSON(c, &body) {
			return
		}
		upload, err := uploader.PresignImageUpload(c.Request.Context(), body.ContentType, body.Size)
		if writeInvalid(c, err) {
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}
		c.JSON(http.StatusOK, uploadResultJSON{
			URL:       upload.URL,
			Headers:   upload.Headers,
			PublicURL: upload.PublicURL,
		})
	}
}

func (b adminWriteJSON) input() admin.Input {
	return admin.Input{
		Slug:        b.Slug,
		Title:       b.Title,
		Summary:     b.Summary,
		BodyMD:      b.BodyMD,
		PublishedAt: b.PublishedAt,
		TagNames:    b.Tags,
	}
}

func postID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, codeInvalidRequest, "invalid id")
		return uuid.UUID{}, false
	}
	return id, true
}

func toAdminPost(p admin.Post) adminPostJSON {
	tagList := make([]tagJSON, len(p.Tags))
	for i, t := range p.Tags {
		tagList[i] = tagJSON(t)
	}
	return adminPostJSON{
		ID:          p.ID,
		Slug:        p.Slug,
		Title:       p.Title,
		Summary:     p.Summary,
		BodyMD:      p.BodyMD,
		PublishedAt: utcPtr(p.PublishedAt),
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
		State:       p.State,
		Tags:        tagList,
	}
}

func writeAdminError(c *gin.Context, err error) {
	if writeInvalid(c, err) {
		return
	}
	switch {
	case errors.Is(err, admin.ErrNotFound):
		writeError(c, http.StatusNotFound, codeNotFound, "post not found")
	case errors.Is(err, admin.ErrConflict):
		writeError(c, http.StatusConflict, codeConflict, "post was modified")
	case errors.Is(err, admin.ErrSlugTaken):
		writeError(c, http.StatusConflict, codeSlugTaken, "slug taken")
	default:
		internalError(c, err)
	}
}
