package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/elmerred09/blog/internal/platform/db/pgconfig"
)

// Error codes in the JSON error body. Clients switch on these, not on message.
const (
	codeInvalidRequest   = "invalid_request"
	codeNotFound         = "not_found"
	codeMethodNotAllowed = "method_not_allowed"
	codeInternal         = "internal"
	codeTimeout          = "timeout"
)

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError aborts the request with a JSON error body.
func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// internalError records err for the request log and responds with a generic
// 500, or a 503 if the request ran out of time, so internal details never
// reach the client.
func internalError(c *gin.Context, err error) {
	_ = c.Error(err)
	if isTimeout(err) {
		writeError(c, http.StatusServiceUnavailable, codeTimeout, "request timed out")
		return
	}
	writeError(c, http.StatusInternalServerError, codeInternal, "internal server error")
}

// isTimeout reports whether err means the request ran out of time: the context
// deadline passed, or Postgres canceled the statement (on the deadline's cancel
// request, or by statement_timeout).
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgconfig.QueryCanceled
}
