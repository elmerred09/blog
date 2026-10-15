package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/search"
)

const DefaultRequestTimeout = 10 * time.Second

type Deps struct {
	Logger *slog.Logger
	DB     Pinger
	Posts  *post.Service
	Search search.Searcher
	// RequestTimeout bounds each request's context. Keep it below the
	// server's WriteTimeout, or the timeout response can't be written.
	RequestTimeout time.Duration
}

func New(d Deps) http.Handler {
	return newEngine(d)
}

func newEngine(d Deps) *gin.Engine {
	r := gin.New()
	// No proxy is trusted until one exists, so X-Forwarded-For can't spoof
	// ClientIP. With a nil list this never returns an error.
	_ = r.SetTrustedProxies(nil)

	timeout := d.RequestTimeout
	if timeout <= 0 {
		timeout = DefaultRequestTimeout
	}

	// requestLogger wraps recovery so a recovered panic is logged as a 500.
	r.Use(requestID(), requestLogger(d.Logger), recovery(d.Logger), requestTimeout(timeout))

	r.GET("/livez", livez)
	r.GET("/readyz", readyz(d.DB))

	v1 := r.Group("/api/v1")
	v1.GET("/posts", listPosts(d.Posts))
	v1.GET("/posts/:slug", getPost(d.Posts))
	v1.GET("/search", searchPosts(d.Search))

	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, codeNotFound, "route not found")
	})
	// Without this, a known path with the wrong method falls through to NoRoute.
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
	})
	return r
}
