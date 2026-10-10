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

func health(db Pinger) gin.HandlerFunc {
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
