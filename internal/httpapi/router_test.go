package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/elmerred09/blog/internal/platform/db/pgconfig"
)

func TestLivez(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	pool.Close() // liveness must not depend on the database
	log, _ := testLogger()

	rec := get(t, New(Deps{Logger: log, DB: pool}), "/livez")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even with the database down; body %s", rec.Code, rec.Body)
	}
	if got := decode[healthBody](t, rec); got.Status != "ok" {
		t.Errorf("status field = %q, want ok", got.Status)
	}
}

func TestReadyz(t *testing.T) {
	t.Parallel()

	t.Run("database reachable", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		log, _ := testLogger()

		rec := get(t, New(Deps{Logger: log, DB: pool}), "/readyz")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
		}
		if got := decode[healthBody](t, rec); got.Status != "ok" {
			t.Errorf("status field = %q, want ok", got.Status)
		}
	})

	t.Run("database unreachable", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		pool.Close()
		log, buf := testLogger()

		rec := get(t, New(Deps{Logger: log, DB: pool}), "/readyz")

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503; body %s", rec.Code, rec.Body)
		}
		if got := decode[healthBody](t, rec); got.Status != "unavailable" {
			t.Errorf("status field = %q, want unavailable", got.Status)
		}
		if line := requestLine(t, buf); line["error"] != "closed pool" {
			t.Errorf("logged error = %q, want %q", line["error"], "closed pool")
		}
	})
}

func TestUnknownRouteIsJSON404(t *testing.T) {
	t.Parallel()
	log, buf := testLogger()

	rec := get(t, New(Deps{Logger: log}), "/nope")

	wantError(t, rec, http.StatusNotFound, codeNotFound)
	line := requestLine(t, buf)
	if line["route"] != "unmatched" || line["level"] != "WARN" {
		t.Errorf("log line = %v, want route unmatched at WARN", line)
	}
}

func TestWrongMethodIsJSON405(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method, target string
	}{
		{http.MethodPost, "/api/v1/posts"},
		{http.MethodDelete, "/api/v1/posts/some-slug"},
		{http.MethodPut, "/livez"},
		{http.MethodPost, "/readyz"},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			t.Parallel()
			log, buf := testLogger()

			rec := serve(t, New(Deps{Logger: log}), httptest.NewRequestWithContext(t.Context(), tc.method, tc.target, nil))

			wantError(t, rec, http.StatusMethodNotAllowed, codeMethodNotAllowed)
			if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
				t.Errorf("Allow = %q, want %q", allow, http.MethodGet)
			}
			if line := requestLine(t, buf); line["level"] != "WARN" {
				t.Errorf("log line = %v, want WARN", line)
			}
		})
	}

	t.Run("unknown path is still 404", func(t *testing.T) {
		t.Parallel()
		log, _ := testLogger()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/nope", nil)
		wantError(t, serve(t, New(Deps{Logger: log}), req), http.StatusNotFound, codeNotFound)
	})
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		incoming string
		keep     bool
	}{
		{"absent is generated", "", false},
		{"valid is echoed", "abc-123_x.y", true},
		{"newline is replaced", "abc\nINFO forged", false},
		{"too long is replaced", strings.Repeat("a", maxRequestIDLen+1), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			log, buf := testLogger()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/nope", nil)
			if tc.incoming != "" {
				req.Header.Set(requestIDHeader, tc.incoming)
			}

			rec := serve(t, New(Deps{Logger: log}), req)

			got := rec.Header().Get(requestIDHeader)
			if tc.keep {
				if got != tc.incoming {
					t.Errorf("X-Request-ID = %q, want %q echoed", got, tc.incoming)
				}
			} else if _, err := uuid.Parse(got); err != nil {
				t.Errorf("X-Request-ID = %q, want a generated uuid", got)
			}
			if line := requestLine(t, buf); line["request_id"] != got {
				t.Errorf("logged request_id = %v, want %q", line["request_id"], got)
			}
		})
	}
}

func TestRequestLogUsesRoutePattern(t *testing.T) {
	t.Parallel()
	log, buf := testLogger()
	r := newEngine(Deps{Logger: log})
	r.GET("/things/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	get(t, r, "/things/42")

	line := requestLine(t, buf)
	want := map[string]any{"method": "GET", "route": "/things/:id", "status": float64(204), "level": "INFO"}
	for k, v := range want {
		if line[k] != v {
			t.Errorf("log %s = %v, want %v (line %v)", k, line[k], v, line)
		}
	}
}

func TestRequestLogErrorField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		errs []string
		want any // nil means the field is absent
	}{
		{"no errors", nil, nil},
		{"one error", []string{"list posts: closed pool"}, "list posts: closed pool"},
		{"several errors", []string{"first", "second"}, "first; second"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			log, buf := testLogger()
			r := newEngine(Deps{Logger: log})
			r.GET("/errs", func(c *gin.Context) {
				for _, msg := range tc.errs {
					_ = c.Error(errors.New(msg))
				}
				c.Status(http.StatusNoContent)
			})

			get(t, r, "/errs")

			if got := requestLine(t, buf)["error"]; got != tc.want {
				t.Errorf("logged error = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestRequestTimeout(t *testing.T) {
	t.Parallel()
	base, _ := env.NewDB(t)
	cfg := base.Config()
	pgconfig.Configure(cfg, 12*time.Second)
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	log, buf := testLogger()
	r := newEngine(Deps{Logger: log, DB: pool, RequestTimeout: 100 * time.Millisecond})
	r.GET("/slow", func(c *gin.Context) {
		if _, err := pool.Exec(c.Request.Context(), "SELECT pg_sleep(5)"); err != nil {
			internalError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	start := time.Now()
	rec := get(t, r, "/slow")
	elapsed := time.Since(start)

	wantError(t, rec, http.StatusServiceUnavailable, codeTimeout)
	if elapsed > 2*time.Second {
		t.Errorf("request took %v; the 100ms deadline didn't stop the query", elapsed)
	}
	const wantLog = "ERROR: canceling statement due to user request (SQLSTATE 57014)"
	if line := requestLine(t, buf); line["error"] != wantLog || line["level"] != "ERROR" {
		t.Errorf("request log line = %v, want error %q at ERROR", line, wantLog)
	}

	// The sleep should stop in Postgres too, not run out its 5 seconds. The
	// connection survives the cancel, so its last query stays in
	// pg_stat_activity; only an active one counts.
	deadline := time.Now().Add(2 * time.Second)
	for {
		var running int
		err := pool.QueryRow(t.Context(),
			`SELECT count(*) FROM pg_stat_activity WHERE query = 'SELECT pg_sleep(5)' AND state = 'active'`).Scan(&running)
		if err != nil {
			t.Fatalf("read pg_stat_activity: %v", err)
		}
		if running == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("pg_sleep still running in Postgres after the request timed out")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestIsTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"deadline exceeded", fmt.Errorf("list posts: %w", context.DeadlineExceeded), true},
		{"statement canceled", fmt.Errorf("list posts: %w", &pgconn.PgError{Code: pgconfig.QueryCanceled}), true},
		{"other Postgres error", &pgconn.PgError{Code: "23505"}, false},
		{"client went away", context.Canceled, false},
		{"plain error", errors.New("boom"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTimeout(tc.err); got != tc.want {
				t.Errorf("isTimeout(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestRecovery(t *testing.T) {
	t.Parallel()
	log, buf := testLogger()
	r := newEngine(Deps{Logger: log})
	r.GET("/boom", func(*gin.Context) { panic("secret detail") })

	rec := get(t, r, "/boom")

	wantError(t, rec, http.StatusInternalServerError, codeInternal)
	if strings.Contains(rec.Body.String(), "secret detail") {
		t.Errorf("response leaks the panic value: %s", rec.Body)
	}

	id := rec.Header().Get(requestIDHeader)
	var sawPanic bool
	for _, l := range logLines(t, buf) {
		if l["msg"] == "panic" {
			sawPanic = true
			if l["panic"] != "secret detail" || l["request_id"] != id || l["stack"] == "" {
				t.Errorf("panic log line = %v, want the value, request id %q, and a stack", l, id)
			}
		}
	}
	if !sawPanic {
		t.Errorf("no panic log line:\n%s", buf)
	}
	if line := requestLine(t, buf); line["status"] != float64(500) || line["level"] != "ERROR" || line["error"] != "panic: secret detail" {
		t.Errorf("request log line = %v, want status 500 at ERROR with error %q", line, "panic: secret detail")
	}
}
