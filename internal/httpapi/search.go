package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/search"
	"github.com/elmerred09/blog/internal/validation"
)

type fragmentJSON struct {
	Text  string `json:"text"`
	Match bool   `json:"match"`
}

type searchResultJSON struct {
	listItemJSON
	Snippet []fragmentJSON `json:"snippet"`
}

type searchResultsJSON struct {
	Items []searchResultJSON `json:"items"`
}

func searchPosts(searcher search.Searcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := search.Query{Text: c.Query("q"), Limit: search.DefaultLimit}
		if raw, ok := c.GetQuery("limit"); ok {
			n, err := strconv.Atoi(raw)
			if err != nil {
				writeInvalid(c, validation.New(validation.Problem{Field: "limit", Message: search.LimitMessage}))
				return
			}
			q.Limit = n
		}

		if writeInvalid(c, q.Validate()) {
			return
		}

		results, err := searcher.Search(c.Request.Context(), q)
		if writeInvalid(c, err) {
			return
		}
		if err != nil {
			internalError(c, err)
			return
		}

		resp := searchResultsJSON{Items: make([]searchResultJSON, len(results))}
		for i, r := range results {
			snippet := make([]fragmentJSON, len(r.Snippet))
			for j, f := range r.Snippet {
				snippet[j] = fragmentJSON(f)
			}
			resp.Items[i] = searchResultJSON{listItemJSON: toListItemJSON(r.ListItem), Snippet: snippet}
		}
		c.JSON(http.StatusOK, resp)
	}
}
