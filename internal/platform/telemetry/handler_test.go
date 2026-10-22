package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestWithTraceAddsIDsOnlyForAValidSpan(t *testing.T) {
	t.Parallel()

	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	span.End()

	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{"no span", context.Background(), false},
		{"valid span", ctx, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := slog.New(WithTrace(slog.NewJSONHandler(&buf, nil)))
			log.InfoContext(tc.ctx, "hello")

			var line map[string]any
			if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
				t.Fatalf("log line: %v\n%s", err, buf.String())
			}
			traceID, spanID := line["trace_id"], line["span_id"]
			if !tc.want {
				if traceID != nil || spanID != nil {
					t.Errorf("trace_id = %v, span_id = %v; want both absent", traceID, spanID)
				}
				return
			}
			sc := span.SpanContext()
			if traceID != sc.TraceID().String() || spanID != sc.SpanID().String() {
				t.Errorf("trace_id = %v, span_id = %v; want %s and %s", traceID, spanID, sc.TraceID(), sc.SpanID())
			}
		})
	}
}
