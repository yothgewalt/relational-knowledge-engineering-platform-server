package middleware

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/telemetry"
)

type ResponseWrapper struct {
	Data    any    `json:"data"`
	TraceID string `json:"trace_id"`
}

func NewResponseWrapperMiddleware(telemetryService telemetry.TelemetryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !telemetryService.IsEnabled() {
			return c.Next()
		}

		err := c.Next()

		ctx := c.Context()
		traceID := telemetry.GetTraceIDFromContext(ctx)

		if traceID == "" {
			return err
		}

		contentType := string(c.Response().Header.ContentType())
		if !strings.Contains(strings.ToLower(contentType), "application/json") {
			if traceID != "" {
				c.Set("X-Trace-ID", traceID)
			}
			return err
		}

		body := c.Response().Body()
		if len(body) == 0 {
			return err
		}

		var originalData any
		if jsonErr := json.Unmarshal(body, &originalData); jsonErr != nil {
			if traceID != "" {
				c.Set("X-Trace-ID", traceID)
			}
			return err
		}

		wrapper := ResponseWrapper{
			Data:    originalData,
			TraceID: traceID,
		}

		return c.JSON(wrapper)
	}
}
