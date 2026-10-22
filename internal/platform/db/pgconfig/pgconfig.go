package pgconfig

import (
	"strconv"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
)

// QueryCanceled is the SQLSTATE Postgres returns when a statement is canceled,
// whether by a cancel request or by statement_timeout.
const QueryCanceled = "57014"

// cancelDeadlineDelay is how long pgx waits for Postgres to act on a cancel
// request before giving up on the connection and closing it.
const cancelDeadlineDelay = time.Second

// Configure makes a context that ends mid-query cancel the statement in
// Postgres and keep the connection, instead of pgx's default of closing it.
// The query then fails with QueryCanceled rather than a context error.
// statementTimeout is a server-side backstop for queries run without a
// deadline; keep it above the request timeout so the request deadline fires
// first.
//
// tp records a span per query, named from sqlc's "-- name:" comment. Nil
// leaves queries untraced. Arguments are not attributes: they can carry
// search terms and post content. otelpgx only emits a span when ctx already
// has a recording parent, so a query joins the request span rather than
// starting a trace of its own.
func Configure(cfg *pgxpool.Config, statementTimeout time.Duration, tp trace.TracerProvider) {
	cfg.ConnConfig.BuildContextWatcherHandler = func(c *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{
			Conn:          c,
			DeadlineDelay: cancelDeadlineDelay,
		}
	}
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(statementTimeout.Milliseconds(), 10)
	if tp != nil {
		cfg.ConnConfig.Tracer = otelpgx.NewTracer(
			otelpgx.WithTracerProvider(tp),
			otelpgx.WithSpanNameFunc(sqlcSpanName),
		)
	}
}
