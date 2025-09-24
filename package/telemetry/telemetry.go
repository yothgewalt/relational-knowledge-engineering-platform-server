package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type TelemetryConfig struct {
	ServiceName    string  `json:"service_name"`
	ServiceVersion string  `json:"service_version"`
	Environment    string  `json:"environment"`
	Enabled        bool    `json:"enabled"`
	JaegerEndpoint string  `json:"jaeger_endpoint"`
	OTLPEndpoint   string  `json:"otlp_endpoint"`
	SamplingRatio  float64 `json:"sampling_ratio"`
	ExporterType   string  `json:"exporter_type"`
}

type HealthStatus struct {
	Enabled       bool          `json:"enabled"`
	ServiceName   string        `json:"service_name"`
	ExporterType  string        `json:"exporter_type"`
	SamplingRatio float64       `json:"sampling_ratio"`
	Error         string        `json:"error,omitempty"`
	Latency       time.Duration `json:"latency"`
}

type TelemetryService interface {
	GetTracer(name string) otelTrace.Tracer
	GetTextMapPropagator() propagation.TextMapPropagator
	Shutdown(ctx context.Context) error
	HealthCheck(ctx context.Context) HealthStatus
	IsEnabled() bool
}

type TelemetryClient struct {
	config         TelemetryConfig
	tracerProvider *trace.TracerProvider
	shutdown       func(context.Context) error
	mu             sync.RWMutex
	initialized    bool
	initError      error
}

func NewTelemetryService(config TelemetryConfig) (*TelemetryClient, error) {
	client := &TelemetryClient{
		config: config,
	}

	if !config.Enabled {
		client.initialized = true
		return client, nil
	}

	if err := client.initialize(); err != nil {
		client.initError = err
		return client, fmt.Errorf("failed to initialize telemetry service: %w", err)
	}

	client.initialized = true
	return client, nil
}

func (t *TelemetryClient) initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(t.config.ServiceName),
			semconv.ServiceVersionKey.String(t.config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(t.config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter trace.SpanExporter
	switch t.config.ExporterType {
	case "jaeger":
		exporter, err = t.createJaegerExporter()
	case "otlp":
		exporter, err = t.createOTLPExporter(ctx)
	case "console":
		return fmt.Errorf("console exporter not implemented yet")
	default:
		return fmt.Errorf("unsupported exporter type: %s", t.config.ExporterType)
	}

	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	samplerOption := trace.WithSampler(trace.TraceIDRatioBased(t.config.SamplingRatio))

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		samplerOption,
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.tracerProvider = tp
	t.shutdown = tp.Shutdown

	return nil
}

func (t *TelemetryClient) createJaegerExporter() (trace.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(t.config.JaegerEndpoint)))
}

func (t *TelemetryClient) createOTLPExporter(ctx context.Context) (trace.SpanExporter, error) {
	return otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(t.config.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
}

func (t *TelemetryClient) GetTracer(name string) otelTrace.Tracer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.config.Enabled || !t.initialized || t.tracerProvider == nil {
		return otel.GetTracerProvider().Tracer(name)
	}

	return t.tracerProvider.Tracer(name)
}

func (t *TelemetryClient) GetTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

func (t *TelemetryClient) IsEnabled() bool {
	return t.config.Enabled
}

func (t *TelemetryClient) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.config.Enabled || t.shutdown == nil {
		return nil
	}

	return t.shutdown(ctx)
}

func (t *TelemetryClient) HealthCheck(ctx context.Context) HealthStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	start := time.Now()
	status := HealthStatus{
		Enabled:       t.config.Enabled,
		ServiceName:   t.config.ServiceName,
		ExporterType:  t.config.ExporterType,
		SamplingRatio: t.config.SamplingRatio,
		Latency:       time.Since(start),
	}

	if !t.config.Enabled {
		return status
	}

	if t.initError != nil {
		status.Error = t.initError.Error()
		return status
	}

	if !t.initialized {
		status.Error = "telemetry service not initialized"
		return status
	}

	tracer := t.GetTracer("health-check")
	_, span := tracer.Start(ctx, "health-check-span")
	span.End()

	status.Latency = time.Since(start)
	return status
}

func StartSpan(ctx context.Context, tracer otelTrace.Tracer, spanName string) (context.Context, otelTrace.Span) {
	return tracer.Start(ctx, spanName)
}

func AddSpanAttributes(span otelTrace.Span, attributes map[string]any) {
	for key, value := range attributes {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
		}
	}
}

func RecordError(span otelTrace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("error", true))
	}
}

func GetTraceIDFromContext(ctx context.Context) string {
	spanCtx := otelTrace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}
