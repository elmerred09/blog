package post_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

// env is shared by every test in this package; nil under -short.
var env *dbtest.Env

func TestMain(m *testing.M) { dbtest.Main(m, &env) }
