package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	t.Run("database reachable", func(t *testing.T) {
		t.Parallel()
		pool, _ := env.NewDB(t)
		log, _ := testLogger()

		rec := get(t, New(Deps{Logger: log, DB: pool}), "/healthz")

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

		rec := get(t, New(Deps{Logger: log, DB: pool}), "/healthz")

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503; body %s", rec.Code, rec.Body)
		}
		if got := decode[healthBody](t, rec); got.Status != "unavailable" {
			t.Errorf("status field = %q, want unavailable", got.Status)
		}
		if line := requestLine(t, buf); line["error"] == nil {
			t.Errorf("request log line has no error: %v", line)
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

			rec := serve(New(Deps{Logger: log}), req)

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
	if line := requestLine(t, buf); line["status"] != float64(500) || line["level"] != "ERROR" {
		t.Errorf("request log line = %v, want status 500 at ERROR", line)
	}
}
