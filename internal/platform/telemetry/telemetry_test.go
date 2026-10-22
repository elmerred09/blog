package telemetry

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestSetupWithoutEndpointIsNoop(t *testing.T) {
	t.Parallel()
	tel, err := Setup(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	t.Cleanup(func() {
		if err := tel.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	})

	if tel.Enabled() {
		t.Error("Enabled = true, want false")
	}
	if tel.TracerProvider() == nil || tel.LoggerProvider() == nil || tel.Propagator() == nil {
		t.Fatal("Setup returned a nil provider")
	}

	_, span := tel.TracerProvider().Tracer("test").Start(context.Background(), "noop")
	defer span.End()
	if span.SpanContext().IsValid() {
		t.Error("no-op span has a valid context; it would be exported")
	}
	if tel.LoggerProvider().Logger("test").Enabled(context.Background(), log.EnabledParameters{Severity: log.SeverityInfo}) {
		t.Error("no-op logger is enabled; it would export")
	}
}

func TestResourceEnvOverridesDefaults(t *testing.T) {
	// t.Setenv refuses a parallel test.
	tests := []struct {
		name     string
		svc      string
		attrs    string
		wantName string
		wantEnv  string // empty means the attribute is absent
	}{
		{"defaults", "", "", ServiceName, ""},
		{"service name", "renamed", "", "renamed", ""},
		{"resource attributes", "", "deployment.environment=prod,service.name=from-attrs", "from-attrs", "prod"},
		{"service name wins over attributes", "renamed", "service.name=from-attrs,deployment.environment=prod", "renamed", "prod"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OTEL_SERVICE_NAME", tc.svc)
			t.Setenv("OTEL_RESOURCE_ATTRIBUTES", tc.attrs)

			res, err := telemetryResource(context.Background())
			if err != nil {
				t.Fatalf("resource: %v", err)
			}
			if got, ok := resourceAttr(res, "service.name"); !ok || got != tc.wantName {
				t.Errorf("service.name = %q, %v; want %q", got, ok, tc.wantName)
			}
			got, ok := resourceAttr(res, "deployment.environment")
			if tc.wantEnv == "" {
				if ok {
					t.Errorf("deployment.environment = %q; want it absent", got)
				}
			} else if !ok || got != tc.wantEnv {
				t.Errorf("deployment.environment = %q, %v; want %q", got, ok, tc.wantEnv)
			}
			if ver, ok := resourceAttr(res, "service.version"); !ok || ver == "" {
				t.Error("service.version is empty")
			}
		})
	}
}

func resourceAttr(res *resource.Resource, key string) (string, bool) {
	for _, kv := range res.Attributes() {
		if string(kv.Key) == key {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}
