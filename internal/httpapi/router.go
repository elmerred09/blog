package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/auth"
	"github.com/elmerred09/blog/internal/media"
	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/search"
	"github.com/elmerred09/blog/internal/tags"
)

const (
	DefaultRequestTimeout = 10 * time.Second
	// maxJSONBody is above the 1 MB markdown limit: JSON quoting and the
	// other fields push a body that is just under the limit over 1 MB, and
	// that post should get the validation message, not "too large".
	maxJSONBody = 2 << 20
)

type Deps struct {
	Logger *slog.Logger
	DB     Pinger
	Posts  *post.Service
	Search search.Searcher
	// Tags serves the public tag list and, with Admin, the editor's.
	Tags *tags.Store
	// Admin is nil when the process has no admin config. The public routes
	// still exist; the admin routes are not registered.
	Admin *Admin
	// RequestTimeout bounds each request's context. Keep it below the
	// server's WriteTimeout, or the timeout response can't be written.
	RequestTimeout time.Duration
}

// Admin is the editor's dependencies. All of them are required together.
type Admin struct {
	Verifier *auth.Verifier
	Posts    *admin.Service
	Media    *media.Uploader
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
	v1.GET("/tags", listTags(d.Tags, tags.Published))

	if d.Admin != nil {
		g := v1.Group("/admin", requireAdmin(d.Admin.Verifier), limitBody(maxJSONBody))
		g.GET("/posts", listAdminPosts(d.Admin.Posts))
		g.POST("/posts", createAdminPost(d.Admin.Posts))
		g.GET("/posts/:id", getAdminPost(d.Admin.Posts))
		g.PUT("/posts/:id", updateAdminPost(d.Admin.Posts))
		g.POST("/preview", previewPost(d.Admin.Posts))
		g.POST("/uploads", createUpload(d.Admin.Media))
		g.GET("/tags", listTags(d.Tags, tags.All))
	}

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
