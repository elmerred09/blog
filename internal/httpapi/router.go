package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/auth"
	"github.com/elmerred09/blog/internal/media"
	"github.com/elmerred09/blog/internal/platform/telemetry"
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
	// TracerProvider records one span per request. Nil uses a no-op provider,
	// so a process with no collector and the tests export nothing. Passed in,
	// rather than read from the global provider.
	TracerProvider trace.TracerProvider
	// Propagator reads an incoming traceparent. Nil uses W3C TraceContext
	// and Baggage.
	Propagator propagation.TextMapPropagator
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

	// The span is first, so the request id and the log line sit on it.
	// requestLogger wraps recovery so a recovered panic is logged as a 500.
	r.Use(
		otelgin.Middleware(telemetry.ServiceName,
			otelgin.WithTracerProvider(tracerProvider(d.TracerProvider)),
			otelgin.WithPropagators(propagator(d.Propagator)),
			otelgin.WithFilter(traceRequest),
			otelgin.WithSpanNameFormatter(spanName),
		),
		requestID(),
		requestLogger(d.Logger),
		recovery(d.Logger),
		requestTimeout(timeout),
	)

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

func tracerProvider(p trace.TracerProvider) trace.TracerProvider {
	if p == nil {
		return tracenoop.NewTracerProvider()
	}
	return p
}

func propagator(p propagation.TextMapPropagator) propagation.TextMapPropagator {
	if p == nil {
		return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	return p
}

// traceRequest drops health probes. A platform hits them constantly, and a
// trace of one says nothing about a slow page.
func traceRequest(r *http.Request) bool {
	switch r.URL.Path {
	case "/livez", "/readyz":
		return false
	default:
		return true
	}
}

// spanName is the route pattern (/api/v1/posts/:slug), the same string the
// request log calls "route", so a span and a log line name the same endpoint.
func spanName(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return "unmatched"
}
