package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/admin"
	"github.com/elmerred09/blog/internal/auth"
	"github.com/elmerred09/blog/internal/config"
	"github.com/elmerred09/blog/internal/httpapi"
	"github.com/elmerred09/blog/internal/media"
	"github.com/elmerred09/blog/internal/platform/db/pgconfig"
	"github.com/elmerred09/blog/internal/post"
	"github.com/elmerred09/blog/internal/render"
	"github.com/elmerred09/blog/internal/search"
	"github.com/elmerred09/blog/internal/tags"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	// requestTimeout is shorter than writeTimeout, so a request that runs out
	// of time still has a few seconds to write its 503.
	requestTimeout = 10 * time.Second
	// statementTimeout is above requestTimeout, so a slow query hits the
	// request deadline first; Postgres enforces it for anything run without one.
	statementTimeout = 12 * time.Second
	shutdownGrace    = 10 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	if err := run(ctx, stop, log); err != nil {
		log.Error("api stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// run serves until ctx is canceled, then shuts down gracefully. stopSignals
// restores default signal handling once shutdown starts, so a second Ctrl-C
// kills the process instead of waiting out the grace period.
func run(ctx context.Context, stopSignals func(), log *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	pgconfig.Configure(poolCfg, statementTimeout)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	deps := httpapi.Deps{
		Logger:         log,
		DB:             pool,
		Posts:          post.NewService(pool),
		Search:         search.NewPostgres(pool),
		Tags:           tags.NewStore(pool),
		RequestTimeout: requestTimeout,
	}
	if cfg.Admin != nil {
		verifier, err := auth.NewVerifier(cfg.Admin.PublicKey, cfg.Admin.Subjects)
		if err != nil {
			return fmt.Errorf("admin verifier: %w", err)
		}
		uploader, err := media.New(cfg.Admin.S3)
		if err != nil {
			return fmt.Errorf("media: %w", err)
		}
		deps.Admin = &httpapi.Admin{
			Verifier: verifier,
			Posts:    admin.NewService(pool, render.New()),
			Media:    uploader,
		}
		log.Info("admin routes enabled")
	} else {
		log.Info("admin routes disabled")
	}

	gin.SetMode(gin.ReleaseMode)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(deps),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("addr", cfg.HTTPAddr))
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	stopSignals()
	log.Info("shutting down", slog.String("grace", shutdownGrace.String()))
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
