package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/tags"
)

type tagsJSON struct {
	Tags []tagJSON `json:"tags"`
}

// listTags serves the public list (tags.Published) and the editor's
// autocomplete (tags.All). Which tags are visible is decided by the store.
func listTags(store *tags.Store, scope tags.Scope) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := store.List(c.Request.Context(), scope)
		if err != nil {
			internalError(c, err)
			return
		}
		out := make([]tagJSON, len(rows))
		for i, row := range rows {
			out[i] = tagJSON(row.Tag)
		}
		c.JSON(http.StatusOK, tagsJSON{Tags: out})
	}
}
