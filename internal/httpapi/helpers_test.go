package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testLogger returns a JSON logger and the buffer it writes to. Handlers log
// synchronously inside ServeHTTP, so the buffer is safe to read afterwards.
func testLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

// logLines decodes each JSON log line.
func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for line := range strings.Lines(buf.String()) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		lines = append(lines, m)
	}
	return lines
}

// requestLine returns the single "request" log line.
func requestLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, l := range logLines(t, buf) {
		if l["msg"] == "request" {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		t.Fatalf("got %d request log lines, want 1:\n%s", len(found), buf)
	}
	return found[0]
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(h, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
}

// decode unmarshals the response body into a T, failing on a non-JSON body.
func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json; body %q", ct, rec.Body)
	}
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	return v
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("status = %d, want %d; body %s", rec.Code, status, rec.Body)
	}
	if got := decode[errorBody](t, rec); got.Error.Code != code || got.Error.Message == "" {
		t.Errorf("error body = %+v, want code %q and a message", got, code)
	}
}
