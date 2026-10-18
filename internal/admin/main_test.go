package admin_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

var env *dbtest.Env

func TestMain(m *testing.M) { dbtest.Main(m, &env) }
