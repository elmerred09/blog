package pgconfig_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/platform/db/pgconfig"
	"github.com/elmerred09/blog/internal/post"
)

// A Get under a parent span records the two sqlc queries as children, and the
// slug passed as an argument stays out of the span attributes.
func TestGetRecordsQuerySpansWithoutArguments(t *testing.T) {
	t.Parallel()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	pool := poolWithTracer(t, tp)

	const secret = "trace-secret-slug"
	dbtest.InsertPost(t, pool, dbtest.WithSlug(secret), dbtest.WithTitle(secret), dbtest.WithTags("go"))

	ctx, parent := tp.Tracer("test").Start(t.Context(), "get")
	got, err := post.NewService(pool).Get(ctx, secret)
	parent.End()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Slug != secret {
		t.Fatalf("slug = %q, want %q", got.Slug, secret)
	}

	// pool.acquire spans sit beside the queries; the queries are the ones named
	// from sqlc. Arguments stay out of every child, including the acquires.
	seen := map[string]bool{"GetPostBySlug": false, "ListTagsForPost": false}
	parentID := parent.SpanContext().SpanID()
	for _, s := range rec.Ended() {
		if s.Parent().SpanID() != parentID {
			continue
		}
		if _, ok := seen[s.Name()]; ok {
			seen[s.Name()] = true
		}
		for _, a := range s.Attributes() {
			if strings.Contains(a.Value.String(), secret) {
				t.Errorf("span %s attribute %s = %s contains the query argument", s.Name(), a.Key, a.Value.String())
			}
		}
	}
	for name, ok := range seen {
		if !ok {
			t.Errorf("no child span named %s", name)
		}
	}
}

func poolWithTracer(t *testing.T, tp *sdktrace.TracerProvider) *pgxpool.Pool {
	t.Helper()
	base, _ := env.NewDB(t)
	cfg := base.Config()
	pgconfig.Configure(cfg, 12*time.Second, tp)
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
