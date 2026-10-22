package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/elmerred09/blog/internal/platform/db/dbtest"
	"github.com/elmerred09/blog/internal/platform/telemetry"
	"github.com/elmerred09/blog/internal/post"
)

func TestTraceRequestSpan(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	dbtest.InsertPost(t, pool, dbtest.WithSlug("traced"))
	tp, rec, h, buf := tracedAPI(t, post.NewService(pool))

	ctx, parent := tp.Tracer("test").Start(t.Context(), "caller")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts/traced", nil)
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(req.Header))
	res := serve(t, h, req)
	parent.End()

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", res.Code, res.Body)
	}
	span := oneSpan(t, rec, "/api/v1/posts/:slug")
	if span.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("parent = %s, want %s from traceparent", span.Parent().SpanID(), parent.SpanContext().SpanID())
	}
	if got, ok := attrInt(span, "http.response.status_code"); !ok || got != http.StatusOK {
		t.Errorf("status attribute = %d, %v; want 200", got, ok)
	}
	if got, ok := attrString(span, "request_id"); !ok || got != res.Header().Get(requestIDHeader) || got == "" {
		t.Errorf("request_id = %q, %v; want the %s response header", got, ok, requestIDHeader)
	}
	line := requestLine(t, buf)
	if line["trace_id"] != span.SpanContext().TraceID().String() {
		t.Errorf("log trace_id = %v, want %s", line["trace_id"], span.SpanContext().TraceID())
	}
}

func TestTraceServerError(t *testing.T) {
	t.Parallel()
	pool, _ := env.NewDB(t)
	pool.Close()
	_, rec, h, _ := tracedAPI(t, post.NewService(pool))

	res := serve(t, h, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/posts", nil))
	wantError(t, res, http.StatusInternalServerError, codeInternal)

	span := oneSpan(t, rec, "/api/v1/posts")
	if span.Status().Code != codes.Error {
		t.Errorf("span status = %v, want Error", span.Status().Code)
	}
}

func TestTraceSkipsLiveness(t *testing.T) {
	t.Parallel()
	_, rec, h, _ := tracedAPI(t, nil)

	res := serve(t, h, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/livez", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if spans := rec.Ended(); len(spans) != 0 {
		t.Errorf("got %d spans for /livez, want 0", len(spans))
	}
}

func tracedAPI(t *testing.T, posts *post.Service) (*sdktrace.TracerProvider, *tracetest.SpanRecorder, http.Handler, *bytes.Buffer) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	base, buf := testLogger()
	log := slog.New(telemetry.WithTrace(base.Handler()))
	h := New(Deps{
		Logger:         log,
		Posts:          posts,
		TracerProvider: tp,
		Propagator:     propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	})
	return tp, rec, h, buf
}

func oneSpan(t *testing.T, rec *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	var found sdktrace.ReadOnlySpan
	for _, s := range rec.Ended() {
		if s.Name() != name {
			continue
		}
		if found != nil {
			t.Fatalf("more than one span named %q", name)
		}
		found = s
	}
	if found == nil {
		var names []string
		for _, s := range rec.Ended() {
			names = append(names, s.Name())
		}
		t.Fatalf("no span named %q; got %v", name, names)
	}
	return found
}

func attrInt(span sdktrace.ReadOnlySpan, key string) (int, bool) {
	for _, a := range span.Attributes() {
		if string(a.Key) == key && a.Value.Type() == attribute.INT64 {
			return int(a.Value.AsInt64()), true
		}
	}
	return 0, false
}

func attrString(span sdktrace.ReadOnlySpan, key string) (string, bool) {
	for _, a := range span.Attributes() {
		if string(a.Key) == key && a.Value.Type() == attribute.STRING {
			return a.Value.AsString(), true
		}
	}
	return "", false
}
