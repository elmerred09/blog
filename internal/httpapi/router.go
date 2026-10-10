package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	Logger *slog.Logger
	DB     Pinger
}

func New(d Deps) http.Handler {
	return newEngine(d)
}

func newEngine(d Deps) *gin.Engine {
	r := gin.New()
	// No proxy is trusted until one exists, so X-Forwarded-For can't spoof
	// ClientIP. With a nil list this never returns an error.
	_ = r.SetTrustedProxies(nil)

	// requestLogger wraps recovery so a recovered panic is logged as a 500.
	r.Use(requestID(), requestLogger(d.Logger), recovery(d.Logger))

	r.GET("/healthz", health(d.DB))

	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, codeNotFound, "route not found")
	})
	return r
}
