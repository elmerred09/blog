package pgconfig

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgxpool"
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
func Configure(cfg *pgxpool.Config, statementTimeout time.Duration) {
	cfg.ConnConfig.BuildContextWatcherHandler = func(c *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{
			Conn:          c,
			DeadlineDelay: cancelDeadlineDelay,
		}
	}
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(statementTimeout.Milliseconds(), 10)
}
