package querytest

import (
	"testing"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
)

// env is shared by every test in this package; nil under -short.
var env *dbtest.Env

func TestMain(m *testing.M) { dbtest.Main(m, &env) }

func TestHarness(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)

	tables := []string{"posts", "tags", "post_tags"}
	for _, table := range tables {
		t.Run(table, func(t *testing.T) {
			var n int
			err := pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&n)
			if err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if n != 0 {
				t.Fatalf("%s has %d rows, want an empty clone of the template", table, n)
			}
		})
	}
}
