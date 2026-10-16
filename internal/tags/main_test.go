package tags_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

// env is shared by the Ensure and Link tests; nil under -short, which skips
// them and still runs the FromNames tests.
var env *dbtest.Env

func TestMain(m *testing.M) { dbtest.Main(m, &env) }
