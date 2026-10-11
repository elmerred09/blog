package pgconfig_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/platform/db/pgconfig"
)

var env *dbtest.Env

func TestMain(m *testing.M) { dbtest.Main(m, &env) }

// configuredPool returns a pool on a fresh test database with Configure applied.
func configuredPool(t *testing.T, statementTimeout time.Duration) *pgxpool.Pool {
	t.Helper()
	base, _ := env.NewDB(t)
	cfg := base.Config()
	pgconfig.Configure(cfg, statementTimeout)
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestStatementTimeoutIsSet(t *testing.T) {
	t.Parallel()
	pool := configuredPool(t, 12*time.Second)

	var got string
	if err := pool.QueryRow(t.Context(), "SHOW statement_timeout").Scan(&got); err != nil {
		t.Fatalf("SHOW statement_timeout: %v", err)
	}
	if got != "12s" {
		t.Errorf("statement_timeout = %q, want 12s", got)
	}
}

func TestStatementTimeoutCancels(t *testing.T) {
	t.Parallel()
	pool := configuredPool(t, 100*time.Millisecond)

	_, err := pool.Exec(t.Context(), "SELECT pg_sleep(5)")

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgconfig.QueryCanceled {
		t.Errorf("err = %v, want SQLSTATE %s from statement_timeout", err, pgconfig.QueryCanceled)
	}
}

func TestExpiredContextCancelsStatementAndKeepsConnection(t *testing.T) {
	t.Parallel()
	pool := configuredPool(t, 12*time.Second)

	conn, err := pool.Acquire(t.Context())
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	pid := conn.Conn().PgConn().PID()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = conn.Exec(ctx, "SELECT pg_sleep(5)")

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != pgconfig.QueryCanceled {
		t.Errorf("err = %v, want SQLSTATE %s (Postgres canceled the statement)", err, pgconfig.QueryCanceled)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Exec returned after %v; the cancel request didn't stop pg_sleep", elapsed)
	}

	if _, err := conn.Exec(t.Context(), "SELECT 1"); err != nil {
		t.Fatalf("connection unusable after cancel: %v", err)
	}
	if conn.Conn().IsClosed() || conn.Conn().PgConn().PID() != pid {
		t.Errorf("connection was replaced; want the same backend (pid %d) reused", pid)
	}
}
