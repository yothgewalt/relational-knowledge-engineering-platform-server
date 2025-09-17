package handlers

import (
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
)

// DocsHandler handles API documentation endpoints
type DocsHandler struct{}

// NewDocsHandler creates a new docs handler
func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

// ServeScalarDocs serves the Scalar API documentation interface
func (h *DocsHandler) ServeScalarDocs(c *fiber.Ctx) error {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - Relational Knowledge Engineering Platform</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
        }
    </style>
</head>
<body>
    <script
        id="api-reference"
        data-url="/docs/openapi.yaml"
        data-configuration='{"theme":"purple","layout":"classic","hideDownloadButton":false,"hideTestRequestButton":false,"isEditable":false,"showSidebar":true,"searchHotKey":"k"}'
    ></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.25.62"></script>
</body>
</html>`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// ServeOpenAPISpec serves the OpenAPI specification file
func (h *DocsHandler) ServeOpenAPISpec(c *fiber.Ctx) error {
	// Try to read from docs directory
	data, err := os.ReadFile("docs/openapi.yaml")
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "OpenAPI specification not found",
		})
	}

	c.Set("Content-Type", "application/yaml")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Content-Type")
	
	return c.Send(data)
}

// GetAPIInfo returns basic API information for the docs
func (h *DocsHandler) GetAPIInfo(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"api": fiber.Map{
			"name":        "Relational Knowledge Engineering Platform API",
			"version":     "1.0.0",
			"description": "A comprehensive API for document processing, knowledge extraction, and graph-based analysis",
		},
		"documentation": fiber.Map{
			"scalar":  "/docs",
			"openapi": "/docs/openapi.yaml",
		},
		"endpoints": fiber.Map{
			"health":     "/health",
			"upload":     "/api/v1/upload/*",
			"documents":  "/api/v1/documents",
			"graphs":     "/api/v1/graphs",
		},
		"features": []string{
			"PDF document upload and processing",
			"Chunked file uploads for large documents", 
			"Text extraction and NLP processing",
			"Knowledge graph generation and analysis",
			"Document management and retrieval",
		},
	})
}