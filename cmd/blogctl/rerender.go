package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/platform/db/pgconfig"
	"github.com/elmerred09/blog/internal/render"
)

const rerenderUsage = "usage: blogctl rerender\n\n" +
	"Renders every post's markdown again and stores the HTML where it changed,\n" +
	"so a renderer change reaches posts saved before it. Reads DATABASE_URL.\n" +
	"updated_at is left alone. Safe to run again, and while the editor is in use.\n"

// rerenderStatementTimeout bounds each query. Every statement is one batch
// read or one single-row update.
const rerenderStatementTimeout = 30 * time.Second

func runRerender(ctx context.Context, args []string, getenv func(string) string, stdout io.Writer) error {
	flags := flag.NewFlagSet("rerender", flag.ContinueOnError)
	flags.Usage = func() { _, _ = fmt.Fprint(flags.Output(), rerenderUsage) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return errors.New("rerender: takes no arguments")
	}
	dsn := getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("rerender: DATABASE_URL is not set")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("rerender: database config: %w", err)
	}
	pgconfig.Configure(cfg, rerenderStatementTimeout, nil)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("rerender: database pool: %w", err)
	}
	defer pool.Close()

	res, err := admin.NewService(pool, render.New()).Rerender(ctx)
	if err != nil {
		return fmt.Errorf("rerender: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "rerender: checked %d posts, updated %d\n", res.Checked, res.Updated)
	return err
}
