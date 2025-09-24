package docs

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
)

func NewHandler() *DocsHandler {
	return NewDocsHandler()
}

type DocsModule struct {
	container.BaseModule
}

func NewDocsModule() *DocsModule {
	base := container.NewBaseModule(
		"docs",
		"1.0.0",
		"Interactive API documentation module using Scalar-Go",
		[]string{},
	)

	return &DocsModule{
		BaseModule: base,
	}
}

func (m *DocsModule) RegisterServices(registry *container.ServiceRegistry) error {
	return nil
}

func (m *DocsModule) RegisterRoutes(router fiber.Router, registry *container.ServiceRegistry) error {
	handler := NewHandler()

	router.Get("/docs", handler.GetDocs)
	router.Get("/docs/*", handler.GetDocs)

	return nil
}

func (m *DocsModule) RegisterMiddleware(registry *container.ServiceRegistry) error {
	return nil
}
