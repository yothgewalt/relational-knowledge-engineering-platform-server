package middleware

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/telemetry"
)

type fiberHeaderCarrier struct {
	ctx *fiber.Ctx
}

func (f *fiberHeaderCarrier) Get(key string) string {
	return f.ctx.Get(key)
}

func (f *fiberHeaderCarrier) Set(key, value string) {
	f.ctx.Set(key, value)
}

func (f *fiberHeaderCarrier) Keys() []string {
	keys := make([]string, 0)

	for k := range f.ctx.Request().Header.All() {
		keys = append(keys, string(k))
	}

	return keys
}

func NewTracingMiddleware(telemetryService telemetry.TelemetryService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !telemetryService.IsEnabled() {
			return c.Next()
		}

		propagator := telemetryService.GetTextMapPropagator()
		carrier := &fiberHeaderCarrier{ctx: c}
		ctx := propagator.Extract(c.Context(), carrier)

		tracer := telemetryService.GetTracer("fiber-http-server")

		spanName := c.Method() + " " + c.Route().Path
		if spanName == " " {
			spanName = c.Method() + " " + c.Path()
		}

		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		c.SetUserContext(ctx)

		span.SetAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.route", c.Route().Path),
			attribute.String("http.target", c.Path()),
			attribute.String("http.scheme", c.Protocol()),
			attribute.String("http.user_agent", c.Get("User-Agent")),
			attribute.Int("http.request_content_length", len(c.Body())),
			attribute.String("http.remote_addr", c.IP()),
		)

		if c.Get("X-Forwarded-For") != "" {
			span.SetAttributes(attribute.String("http.x_forwarded_for", c.Get("X-Forwarded-For")))
		}

		err := c.Next()

		span.SetAttributes(
			attribute.Int("http.response.status_code", c.Response().StatusCode()),
			attribute.Int("http.response_content_length", len(c.Response().Body())),
		)

		if err != nil {
			span.RecordError(err)
			telemetry.RecordError(span, err)
		}

		statusCode := c.Response().StatusCode()
		if statusCode >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}

		propagator.Inject(ctx, carrier)

		return err
	}
}
