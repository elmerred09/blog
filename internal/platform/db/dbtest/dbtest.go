// Package dbtest runs integration tests against a throwaway Postgres in Docker.
//
// Start launches one container per test package (Main wraps it for TestMain) and
// migrates a template database. NewDB gives each test its own database cloned
// from that template, so tests are isolated and can run in parallel.
package dbtest

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	blogdb "github.com/elmerred09/blog/db"
	"github.com/elmerred09/blog/internal/db"
)

const (
	image = "postgres:18"
	// initDB is created by the container's entrypoint. Nothing uses it; it
	// just must not be templateName, which createTemplate creates itself.
	initDB       = "blog"
	adminDB      = "postgres"
	templateName = "blog_template"

	username = "blog"
	password = "blog"

	// maxConnsPerTest keeps parallel tests under Postgres's default
	// max_connections (100); pgxpool defaults to max(4, NumCPU) per pool.
	maxConnsPerTest = 4
)

// Env is a running test Postgres. A nil *Env means database tests are skipped.
type Env struct {
	container *postgres.PostgresContainer
	// admin connects to the "postgres" maintenance database. It's a pool, not
	// a single conn, because parallel tests issue CREATE DATABASE concurrently.
	admin *pgxpool.Pool
	// baseCfg is the container's connection config. Copy it and set Database
	// to connect to the template or a per-test database.
	baseCfg *pgxpool.Config
	// next numbers the per-test databases (test_1, test_2, ...).
	next atomic.Int64
}

// Main is a whole TestMain for packages with database tests:
//
//	var env *dbtest.Env
//
//	func TestMain(m *testing.M) { dbtest.Main(m, &env) }
//
// It starts Postgres, stores the Env in *env, runs the tests, and stops the
// container. Under -short it starts nothing and leaves *env nil, so NewDB
// skips the database tests and the rest still run.
func Main(m *testing.M, env **Env) {
	flag.Parse() // testing.Short needs parsed flags before m.Run
	if testing.Short() {
		m.Run()
		return
	}

	e, stop, err := Start(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "start test database: %v\n", err)
		os.Exit(1)
	}
	defer stop()

	*env = e
	m.Run()
}

// Start runs Postgres in Docker, migrates the template database, and returns
// the Env plus a stop func that closes connections and removes the container.
func Start(ctx context.Context) (_ *Env, _ func(), err error) {
	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase(initDB),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		postgres.BasicWaitStrategies(),
	)

	e := &Env{container: container}

	stop := func() {
		if e.admin != nil {
			e.admin.Close()
		}
		if e.container != nil {
			if err := testcontainers.TerminateContainer(e.container); err != nil {
				log.Printf("testcontainers.TerminateContainer: %v", err)
			}
		}
	}

	defer func() {
		if err != nil {
			stop()
		}
	}()

	if err != nil {
		return nil, nil, fmt.Errorf("postgres.Run: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("container.ConnectionString: %w", err)
	}

	e.baseCfg, err = pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}

	e.admin, err = pgxpool.NewWithConfig(ctx, e.configFor(adminDB))
	if err != nil {
		return nil, nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}

	if err := e.createTemplate(ctx); err != nil {
		return nil, nil, fmt.Errorf("createTemplate: %w", err)
	}

	return e, stop, nil
}

// configFor returns a copy of the base config pointed at database name.
func (e *Env) configFor(name string) *pgxpool.Config {
	cfg := e.baseCfg.Copy()
	cfg.ConnConfig.Database = name
	return cfg
}

// createTemplate creates the template database and runs every migration in it.
func (e *Env) createTemplate(ctx context.Context) error {
	if _, err := e.admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{templateName}.Sanitize()); err != nil {
		return fmt.Errorf("create template database: %w", err)
	}

	migrations, err := fs.Sub(blogdb.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("fs.Sub: %w", err)
	}

	sqlDB := stdlib.OpenDB(*e.configFor(templateName).ConnConfig)
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations)
	if err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("goose.NewProvider: %w", err)
	}
	// Provider.Close also closes sqlDB. No connection to the template may stay
	// open, or CREATE DATABASE ... TEMPLATE in NewDB fails.
	defer func() {
		if err := p.Close(); err != nil {
			log.Printf("goose provider close: %v", err)
		}
	}()

	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	if _, err := e.admin.Exec(ctx, "ALTER DATABASE "+pgx.Identifier{templateName}.Sanitize()+" WITH IS_TEMPLATE true"); err != nil {
		return fmt.Errorf("mark template: %w", err)
	}

	return nil
}

// NewDB returns a pool and queries for a fresh database cloned from the
// template. The database is dropped when the test ends. It skips the test when
// e is nil (tests run with -short).
func (e *Env) NewDB(t testing.TB) (*pgxpool.Pool, *db.Queries) {
	t.Helper()
	if e == nil {
		t.Skip("database tests skipped (-short)")
	}

	name := fmt.Sprintf("test_%d", e.next.Add(1))
	ident := pgx.Identifier{name}.Sanitize()

	create := "CREATE DATABASE " + ident + " TEMPLATE " + pgx.Identifier{templateName}.Sanitize()
	if _, err := e.admin.Exec(t.Context(), create); err != nil {
		t.Fatalf("create test database %s: %v", name, err)
	}
	// Registered before the pool's cleanup, so it runs after it: cleanups run
	// last-in, first-out, and the database can only be dropped once the pool
	// is closed. Background context because t.Context() is canceled by now.
	t.Cleanup(func() {
		if _, err := e.admin.Exec(context.Background(), "DROP DATABASE "+ident+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database %s: %v", name, err)
		}
	})

	cfg := e.configFor(name)
	cfg.MaxConns = maxConnsPerTest
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("connect to test database %s: %v", name, err)
	}
	t.Cleanup(pool.Close)

	return pool, db.New(pool)
}
