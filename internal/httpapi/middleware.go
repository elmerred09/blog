package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
	maxRequestIDLen = 64
)

// requestID reuses the caller's X-Request-ID when it is safe to log, otherwise
// generates one, and echoes it in the response.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !validRequestID(id) {
			id = uuid.NewString()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDHeader, id)
		trace.SpanFromContext(c.Request.Context()).SetAttributes(attribute.String("request_id", id))
		c.Next()
	}
}

// validRequestID accepts short ids of [A-Za-z0-9._-], so a client can't inject
// newlines or huge values into the logs.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		switch {
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9', r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// requestLogger writes one line per request after it completes: 5xx at Error,
// 4xx at Warn, everything else at Info. The route is the registered pattern
// (/api/v1/posts/:slug), so lines group by endpoint rather than by URL.
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			slog.Int("bytes", max(c.Writer.Size(), 0)),
			slog.String("request_id", c.GetString(requestIDKey)),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", strings.Join(c.Errors.Errors(), "; ")))
		}
		log.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}

// requestTimeout puts a deadline on the request context, so database calls made
// with c.Request.Context() give up when it passes. http.Server's WriteTimeout
// doesn't do this: it cuts off the response but leaves the handler running.
func requestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// recovery turns a panic into a JSON 500 and logs the panic value and stack.
// http.ErrAbortHandler is re-panicked: net/http uses it to abort a response
// on purpose.
func recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(rec)
			}

			log.LogAttrs(c.Request.Context(), slog.LevelError, "panic",
				slog.String("panic", fmt.Sprint(rec)),
				slog.String("stack", string(debug.Stack())),
				slog.String("request_id", c.GetString(requestIDKey)),
			)
			if c.Writer.Written() {
				c.Abort()
				return
			}
			internalError(c, fmt.Errorf("panic: %v", rec))
		}()
		c.Next()
	}
}
