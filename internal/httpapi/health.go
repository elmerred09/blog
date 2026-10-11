package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const healthTimeout = 2 * time.Second

// Pinger checks the database connection; *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthBody struct {
	Status string `json:"status"`
}

// livez reports that the process is up and serving. It checks nothing else:
// the platform restarts the process when liveness fails, and a restart can't
// fix a database outage, only cause a restart loop.
func livez(c *gin.Context) {
	c.JSON(http.StatusOK, healthBody{Status: "ok"})
}

// readyz reports whether this instance can serve traffic, which needs the
// database. Failing it only takes the instance out of rotation.
func readyz(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), healthTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusServiceUnavailable, healthBody{Status: "unavailable"})
			return
		}
		c.JSON(http.StatusOK, healthBody{Status: "ok"})
	}
}
