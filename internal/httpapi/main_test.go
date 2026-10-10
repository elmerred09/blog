package httpapi

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

// env is nil under -short, which skips the tests that need Postgres.
var env *dbtest.Env

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	dbtest.Main(m, &env)
}
