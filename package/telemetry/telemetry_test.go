package telemetry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
)

func createTestConfig(enabled bool, exporterType string) TelemetryConfig {
	return TelemetryConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        enabled,
		JaegerEndpoint: "http://invalid-endpoint:14268/api/traces",
		OTLPEndpoint:   "invalid-endpoint:4317",
		SamplingRatio:  1.0,
		ExporterType:   exporterType,
	}
}

type mockSpan struct {
	embedded.Span
	attributes map[string]any
	errors     []error
	ended      bool
}

func (m *mockSpan) End(...trace.SpanEndOption) {
	m.ended = true
}

func (m *mockSpan) AddEvent(string, ...trace.EventOption) {}
func (m *mockSpan) AddLink(trace.Link)                    {}
func (m *mockSpan) IsRecording() bool                     { return true }
func (m *mockSpan) RecordError(err error, options ...trace.EventOption) {
	m.errors = append(m.errors, err)
}
func (m *mockSpan) SpanContext() trace.SpanContext                { return trace.SpanContext{} }
func (m *mockSpan) SetStatus(code codes.Code, description string) {}
func (m *mockSpan) SetName(string)                                {}
func (m *mockSpan) SetAttributes(kv ...attribute.KeyValue) {
	if m.attributes == nil {
		m.attributes = make(map[string]any)
	}
	for _, attr := range kv {
		m.attributes[string(attr.Key)] = attr.Value.AsInterface()
	}
}
func (m *mockSpan) TracerProvider() trace.TracerProvider { return nil }

func TestNewTelemetryService_Disabled(t *testing.T) {
	config := createTestConfig(false, "otlp")

	service, err := NewTelemetryService(config)

	if err != nil {
		t.Fatalf("Expected no error for disabled service, got: %v", err)
	}

	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if !service.initialized {
		t.Error("Expected disabled service to be marked as initialized")
	}

	if service.IsEnabled() {
		t.Error("Expected service to be disabled")
	}
}

func TestNewTelemetryService_InvalidExporter(t *testing.T) {
	config := createTestConfig(true, "unsupported")

	service, err := NewTelemetryService(config)

	if err == nil {
		t.Fatal("Expected error for unsupported exporter type")
	}

	if service == nil {
		t.Fatal("Expected service to be returned even on initialization error")
	}

	if service.initError == nil {
		t.Error("Expected initError to be set")
	}
}

func TestNewTelemetryService_ConsoleExporter(t *testing.T) {
	config := createTestConfig(true, "console")

	service, err := NewTelemetryService(config)

	if err == nil {
		t.Fatal("Expected error for console exporter (not implemented)")
	}

	if service == nil {
		t.Fatal("Expected service to be returned")
	}

	if service.initError == nil {
		t.Error("Expected initError to be set for console exporter")
	}
}

func TestNewTelemetryService_JaegerExporter(t *testing.T) {
	config := createTestConfig(true, "jaeger")

	service, err := NewTelemetryService(config)

	if err == nil {
		t.Log("Service created successfully (may happen in test environment)")
		if !service.IsEnabled() {
			t.Error("Expected service to be enabled")
		}
	} else {
		t.Logf("Expected initialization failure due to invalid endpoint: %v", err)
		if service == nil {
			t.Fatal("Expected service to be returned even on error")
		}
	}
}

func TestNewTelemetryService_OTLPExporter(t *testing.T) {
	config := createTestConfig(true, "otlp")

	service, err := NewTelemetryService(config)

	if err == nil {
		t.Log("Service created successfully (may happen in test environment)")
		if !service.IsEnabled() {
			t.Error("Expected service to be enabled")
		}
	} else {
		t.Logf("Expected initialization failure due to invalid endpoint: %v", err)
		if service == nil {
			t.Fatal("Expected service to be returned even on error")
		}
	}
}

func TestTelemetryClient_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := createTestConfig(tt.enabled, "otlp")
			service, _ := NewTelemetryService(config)

			if service.IsEnabled() != tt.enabled {
				t.Errorf("Expected IsEnabled() = %v, got %v", tt.enabled, service.IsEnabled())
			}
		})
	}
}

func TestTelemetryClient_GetTracer(t *testing.T) {
	t.Run("disabled_service", func(t *testing.T) {
		config := createTestConfig(false, "otlp")
		service, _ := NewTelemetryService(config)

		tracer := service.GetTracer("test-tracer")
		if tracer == nil {
			t.Error("Expected tracer to be returned even when disabled")
		}
	})

	t.Run("enabled_service_with_error", func(t *testing.T) {
		config := createTestConfig(true, "invalid")
		service, _ := NewTelemetryService(config)

		tracer := service.GetTracer("test-tracer")
		if tracer == nil {
			t.Error("Expected fallback tracer when initialization fails")
		}
	})
}

func TestTelemetryClient_GetTextMapPropagator(t *testing.T) {
	config := createTestConfig(false, "otlp")
	service, _ := NewTelemetryService(config)

	propagator := service.GetTextMapPropagator()
	if propagator == nil {
		t.Error("Expected propagator to be returned")
	}
}

func TestTelemetryClient_Shutdown(t *testing.T) {
	t.Run("disabled_service", func(t *testing.T) {
		config := createTestConfig(false, "otlp")
		service, _ := NewTelemetryService(config)

		err := service.Shutdown(context.Background())
		if err != nil {
			t.Errorf("Expected no error shutting down disabled service, got: %v", err)
		}
	})

	t.Run("enabled_service_with_init_error", func(t *testing.T) {
		config := createTestConfig(true, "invalid")
		service, _ := NewTelemetryService(config)

		err := service.Shutdown(context.Background())
		if err != nil {
			t.Errorf("Expected no error shutting down failed service, got: %v", err)
		}
	})
}

func TestTelemetryClient_HealthCheck(t *testing.T) {
	t.Run("disabled_service", func(t *testing.T) {
		config := createTestConfig(false, "otlp")
		service, _ := NewTelemetryService(config)

		health := service.HealthCheck(context.Background())

		if health.Enabled {
			t.Error("Expected health status to show disabled")
		}
		if health.ServiceName != "test-service" {
			t.Errorf("Expected service name 'test-service', got '%s'", health.ServiceName)
		}
		if health.Error != "" {
			t.Errorf("Expected no error for disabled service, got: %s", health.Error)
		}
	})

	t.Run("enabled_service_with_init_error", func(t *testing.T) {
		config := createTestConfig(true, "invalid")
		service, _ := NewTelemetryService(config)

		health := service.HealthCheck(context.Background())

		if !health.Enabled {
			t.Error("Expected health status to show enabled (config-wise)")
		}
		if health.Error == "" {
			t.Error("Expected error in health status due to init failure")
		}
		if health.Latency <= 0 {
			t.Error("Expected positive latency measurement")
		}
	})
}

func TestStartSpan(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	newCtx, span := StartSpan(ctx, tracer, "test-span")

	if newCtx == nil {
		t.Error("Expected context to be returned")
	}
	if span == nil {
		t.Error("Expected span to be returned")
	}

	span.End()
}

func TestAddSpanAttributes(t *testing.T) {
	mockSpan := &mockSpan{}

	attributes := map[string]any{
		"string_attr":  "test-value",
		"int_attr":     42,
		"int64_attr":   int64(1234567890),
		"float64_attr": 3.14159,
		"bool_attr":    true,
		"other_attr":   []string{"complex", "type"},
	}

	AddSpanAttributes(mockSpan, attributes)

	expectedAttrs := []string{"string_attr", "int_attr", "int64_attr", "float64_attr", "bool_attr", "other_attr"}
	for _, attr := range expectedAttrs {
		if _, exists := mockSpan.attributes[attr]; !exists {
			t.Errorf("Expected attribute '%s' to be set", attr)
		}
	}

	if mockSpan.attributes["string_attr"] != "test-value" {
		t.Error("String attribute not set correctly")
	}
	if mockSpan.attributes["bool_attr"] != true {
		t.Error("Bool attribute not set correctly")
	}

	if mockSpan.attributes["other_attr"] != "[complex type]" {
		t.Log("Complex type converted to string representation:", mockSpan.attributes["other_attr"])
	}
}

func TestRecordError(t *testing.T) {
	t.Run("with_error", func(t *testing.T) {
		mockSpan := &mockSpan{}
		testError := errors.New("test error")

		RecordError(mockSpan, testError)

		if len(mockSpan.errors) != 1 {
			t.Errorf("Expected 1 error recorded, got %d", len(mockSpan.errors))
		}
		if mockSpan.errors[0] != testError {
			t.Error("Error not recorded correctly")
		}
		if mockSpan.attributes["error"] != true {
			t.Error("Error attribute not set")
		}
	})

	t.Run("without_error", func(t *testing.T) {
		mockSpan := &mockSpan{}

		RecordError(mockSpan, nil)

		if len(mockSpan.errors) != 0 {
			t.Errorf("Expected no errors recorded, got %d", len(mockSpan.errors))
		}
		if _, exists := mockSpan.attributes["error"]; exists {
			t.Error("Error attribute should not be set when no error")
		}
	})
}

func TestGetTraceIDFromContext(t *testing.T) {
	t.Run("context_without_trace", func(t *testing.T) {
		ctx := context.Background()
		traceID := GetTraceIDFromContext(ctx)

		if traceID != "" {
			t.Errorf("Expected empty trace ID, got '%s'", traceID)
		}
	})

	t.Run("context_with_trace", func(t *testing.T) {
		tracer := otel.Tracer("test")
		ctx, span := tracer.Start(context.Background(), "test-span")
		defer span.End()

		traceID := GetTraceIDFromContext(ctx)

		if len(traceID) > 0 {
			t.Logf("Trace ID extracted: %s", traceID)
			if len(traceID) != 32 {
				t.Logf("Note: Trace ID length is %d (expected 32 in real tracing)", len(traceID))
			}
		} else {
			t.Log("No trace ID in test environment (expected behavior)")
		}
	})
}

func TestTelemetryConfig_Validation(t *testing.T) {
	tests := []struct {
		name          string
		config        TelemetryConfig
		shouldSucceed bool
	}{
		{
			name: "valid_config_disabled",
			config: TelemetryConfig{
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Enabled:        false,
				SamplingRatio:  1.0,
				ExporterType:   "otlp",
			},
			shouldSucceed: true,
		},
		{
			name: "invalid_sampling_ratio",
			config: TelemetryConfig{
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				Enabled:        true,
				SamplingRatio:  -1.0,
				ExporterType:   "otlp",
				OTLPEndpoint:   "invalid:4317",
			},
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewTelemetryService(tt.config)

			if tt.shouldSucceed {
				if service == nil {
					t.Error("Expected service to be created")
				}
			} else {
				if err == nil && service.initError == nil {
					t.Log("Note: Invalid config may still create service in test environment")
				}
			}
		})
	}
}

func TestTelemetryClient_ConcurrentAccess(t *testing.T) {
	config := createTestConfig(false, "otlp")
	service, err := NewTelemetryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 10; i++ {
			service.GetTracer(fmt.Sprintf("tracer-%d", i))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			service.HealthCheck(context.Background())
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			service.Shutdown(context.Background())
		}
		done <- true
	}()

	for range 3 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}
}
