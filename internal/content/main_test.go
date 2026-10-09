package content_test

import (
	"testing"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

// env is shared by the importer tests; nil under -short, which skips them and
// still runs the parse, slug, render, and path tests.
var env *dbtest.Env

func TestMain(m *testing.M) { dbtest.Main(m, &env) }
