// Package telemetry configures OpenTelemetry traces and logs for the API.
// With no endpoint it installs no-op providers, so tests and CI export nothing.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// ServiceName is the resource attribute and the server span's service.
const ServiceName = "blog-api"

// Config selects where telemetry goes. The endpoint is
// OTEL_EXPORTER_OTLP_ENDPOINT (for example http://localhost:4318). Empty
// means don't export. Sampling, headers, timeout, and the resource
// (OTEL_SERVICE_NAME, OTEL_RESOURCE_ATTRIBUTES) come from the standard
// OTEL_* variables. The service name defaults to blog-api; the environment
// overrides it.
type Config struct {
	Endpoint string
}

// Telemetry holds the providers for a process. Shutdown flushes them.
type Telemetry struct {
	traces     trace.TracerProvider
	logs       log.LoggerProvider
	propagator propagation.TextMapPropagator
	shutdown   func(context.Context) error
}

// Setup builds a tracer and a logger provider. Without an endpoint both are
// no-ops and Shutdown does nothing.
func Setup(ctx context.Context, cfg Config) (*Telemetry, error) {
	prop := autoprop.NewTextMapPropagator()
	if cfg.Endpoint == "" {
		return &Telemetry{
			traces:     tracenoop.NewTracerProvider(),
			logs:       lognoop.NewLoggerProvider(),
			propagator: prop,
		}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Exporter startup keeps this context. Detach it from the signal context
	// so cancellation at shutdown doesn't race the explicit Shutdown flush.
	ctx = context.WithoutCancel(ctx)

	res, err := telemetryResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	// The exporters read OTEL_EXPORTER_OTLP_ENDPOINT themselves and append
	// /v1/traces and /v1/logs. Passing the URL in WithEndpointURL would send
	// logs to the host root, which the collector answers with 404.
	traces, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traces),
	)

	logs, err := otlploghttp.New(ctx)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logs)),
	)

	return &Telemetry{
		traces:     tp,
		logs:       lp,
		propagator: prop,
		shutdown: func(ctx context.Context) error {
			var errs []error
			if err := tp.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown traces: %w", err))
			}
			if err := lp.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("shutdown logs: %w", err))
			}
			return errors.Join(errs...)
		},
	}, nil
}

// TracerProvider records spans. It is a no-op when Setup had no endpoint.
func (t *Telemetry) TracerProvider() trace.TracerProvider { return t.traces }

// LoggerProvider records logs. It is a no-op when Setup had no endpoint.
func (t *Telemetry) LoggerProvider() log.LoggerProvider { return t.logs }

// Propagator is W3C TraceContext plus Baggage, unless OTEL_PROPAGATORS says otherwise.
func (t *Telemetry) Propagator() propagation.TextMapPropagator { return t.propagator }

// Enabled reports whether spans and logs are exported.
func (t *Telemetry) Enabled() bool { return t.shutdown != nil }

// Shutdown flushes buffered telemetry. It is safe on a no-op setup.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.shutdown == nil {
		return nil
	}
	return t.shutdown(ctx)
}

// telemetryResource defaults the service name and version, then lets
// OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES override them. WithFromEnv
// is last because resource.New keeps the later value. Inside that detector,
// OTEL_SERVICE_NAME wins over a service.name in the attribute list.
func telemetryResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(serviceVersion()),
		),
		resource.WithFromEnv(),
	)
}

// serviceVersion prefers the module version, then the VCS revision go run stamps
// into the binary. A local `go run` has neither, and reports "devel".
func serviceVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			return s.Value
		}
	}
	return "devel"
}
