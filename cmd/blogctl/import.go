package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/config"
	"github.com/elmerred09/blog/internal/content"
)

const importUsage = "usage: blogctl import PATH [PATH...]\n\n" +
	"Upserts markdown files (or every *.md under a directory) into Postgres.\n" +
	"Paths must be inside the current directory. Requires DATABASE_URL (make import sets it).\n"

func runImport(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.Usage = func() { _, _ = fmt.Fprint(flags.Output(), importUsage) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() == 0 {
		flags.Usage()
		return errors.New("import: no paths given")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}
	paths, err := fsPaths(cwd, flags.Args())
	if err != nil {
		return err
	}

	// A Root, unlike os.DirFS, also refuses symlinks that lead outside cwd.
	root, err := os.OpenRoot(cwd)
	if err != nil {
		return fmt.Errorf("open %s: %w", cwd, err)
	}
	defer func() { _ = root.Close() }()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect to database (is it running? try make up): %w", err)
	}

	summary, err := content.NewImporter(pool, content.NewRenderer()).Import(ctx, root.FS(), paths)
	if err != nil {
		return err
	}
	return printSummary(stdout, summary)
}

func printSummary(w io.Writer, s content.Summary) error {
	// tabwriter buffers until Flush, so write errors surface there.
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "PATH\tSLUG\tOUTCOME")
	for _, f := range s.Files {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\n", f.Path, f.Slug, f.Outcome)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("write summary: %w", err)
	}

	_, err := fmt.Fprintf(w, "\n%d inserted, %d updated, %d unchanged; %d tags created; %d tag links added, %d removed\n",
		s.Count(content.Inserted), s.Count(content.Updated), s.Count(content.Unchanged),
		s.TagsCreated, s.TagLinksAdded, s.TagLinksRemoved)
	if err != nil {
		return fmt.Errorf("write summary: %w", err)
	}
	return nil
}
