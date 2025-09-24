package telemetry

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/middleware"
)

type TelemetryModule struct {
	container.BaseModule
}

func NewTelemetryModule() *TelemetryModule {
	return &TelemetryModule{
		BaseModule: container.NewBaseModule(
			"telemetry",
			"1.0.0",
			"OpenTelemetry tracing and middleware",
			[]string{},
		),
	}
}

func (m *TelemetryModule) RegisterServices(registry *container.ServiceRegistry) error {
	return nil
}

func (m *TelemetryModule) RegisterMiddleware(registry *container.ServiceRegistry) error {
	return nil
}

func (m *TelemetryModule) RegisterRoutes(router fiber.Router, registry *container.ServiceRegistry) error {
	telemetryService := registry.GetTelemetry()
	if telemetryService == nil {
		return nil
	}

	router.Use(middleware.NewTracingMiddleware(telemetryService))
	router.Use(middleware.NewResponseWrapperMiddleware(telemetryService))

	router.Get("/health/telemetry", m.handleTelemetryHealthCheck(registry))

	return nil
}

func (m *TelemetryModule) handleTelemetryHealthCheck(registry *container.ServiceRegistry) fiber.Handler {
	return func(c *fiber.Ctx) error {
		telemetryService := registry.GetTelemetry()
		if telemetryService == nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "service_unavailable",
				"error":  "telemetry service not available",
			})
		}

		health := telemetryService.HealthCheck(c.Context())

		status := 200
		if health.Error != "" {
			status = 503
		}

		return c.Status(status).JSON(health)
	}
}