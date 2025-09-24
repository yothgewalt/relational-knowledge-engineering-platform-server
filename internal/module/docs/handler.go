package docs

import (
	"os"
	"path/filepath"

	scalargo "github.com/bdpiprava/scalar-go"
	"github.com/gofiber/fiber/v2"
)

// Note: Embedded files will be loaded at build time from docs directory

type DocsHandler struct{}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// GetDocs godoc
// @Summary Get API documentation
// @Description Serve interactive API documentation using Scalar
// @Tags documentation
// @Accept html
// @Produce html
// @Success 200 {string} string "HTML documentation page"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /docs [get]
func (h *DocsHandler) GetDocs(c *fiber.Ctx) error {
	cwd, _ := os.Getwd()
	swaggerPath := filepath.Join(cwd, "docs", "swagger.json")

	specBytes, err := os.ReadFile(swaggerPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to load API specification",
			"message": "Make sure swagger.json exists in docs directory",
		})
	}

	html, err := scalargo.NewV2(
		scalargo.WithSpecBytes(specBytes),
		scalargo.WithTheme(scalargo.ThemeDefault),
		scalargo.WithLayout(scalargo.LayoutModern),
		scalargo.WithMetaDataOpts(
			scalargo.WithTitle("🚀 Relational Knowledge Engineering Platform API"),
		),
		scalargo.WithDarkMode(),
		scalargo.WithSidebarVisibility(true),
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to generate documentation",
			"message": err.Error(),
		})
	}

	c.Set("Content-Type", "text/html; charset=utf-8")

	c.Set("Cache-Control", "public, max-age=3600")

	return c.SendString(html)
}
