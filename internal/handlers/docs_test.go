package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocsHandler_ServeScalarDocs(t *testing.T) {
	docsHandler := NewDocsHandler()
	app := fiber.New()
	app.Get("/docs", docsHandler.ServeScalarDocs)

	req, _ := http.NewRequest("GET", "/docs", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

	// Read response body
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	// Check that the HTML contains expected elements
	assert.Contains(t, bodyStr, "<!DOCTYPE html>")
	assert.Contains(t, bodyStr, "API Documentation - Relational Knowledge Engineering Platform")
	assert.Contains(t, bodyStr, "@scalar/api-reference")
	assert.Contains(t, bodyStr, "data-url=\"/docs/openapi.yaml\"")
}

func TestDocsHandler_GetAPIInfo(t *testing.T) {
	docsHandler := NewDocsHandler()
	app := fiber.New()
	app.Get("/", docsHandler.GetAPIInfo)

	req, _ := http.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	// The response should contain API information
	body := make([]byte, 2048)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])

	// Check that the JSON contains expected fields
	assert.Contains(t, bodyStr, "\"api\"")
	assert.Contains(t, bodyStr, "\"documentation\"")
	assert.Contains(t, bodyStr, "\"endpoints\"")
	assert.Contains(t, bodyStr, "\"features\"")
	assert.Contains(t, bodyStr, "Relational Knowledge Engineering Platform API")
	assert.Contains(t, bodyStr, "/docs")
	assert.Contains(t, bodyStr, "/docs/openapi.yaml")
}

func TestDocsHandler_ServeOpenAPISpec(t *testing.T) {
	docsHandler := NewDocsHandler()
	app := fiber.New()
	app.Get("/docs/openapi.yaml", docsHandler.ServeOpenAPISpec)

	req, _ := http.NewRequest("GET", "/docs/openapi.yaml", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// The response status depends on whether the openapi.yaml file exists
	// In CI/test environments, it might return 404, which is acceptable
	if resp.StatusCode == fiber.StatusOK {
		assert.Equal(t, "application/yaml", resp.Header.Get("Content-Type"))
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
		
		// Read response body
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		
		// Check that it's valid YAML content
		assert.True(t, strings.HasPrefix(bodyStr, "openapi:") || strings.Contains(bodyStr, "openapi:"))
	} else {
		// File not found is acceptable in test environment
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	}
}

func TestNewDocsHandler(t *testing.T) {
	handler := NewDocsHandler()
	assert.NotNil(t, handler)
	assert.IsType(t, &DocsHandler{}, handler)
}