package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error codes in the JSON error body. Clients switch on these, not on message.
const (
	codeInvalidRequest = "invalid_request"
	codeNotFound       = "not_found"
	codeInternal       = "internal"
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
// 500, so internal details never reach the client.
func internalError(c *gin.Context, err error) {
	_ = c.Error(err)
	writeError(c, http.StatusInternalServerError, codeInternal, "internal server error")
}
