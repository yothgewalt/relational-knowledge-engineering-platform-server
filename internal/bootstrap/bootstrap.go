package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/handlers"
)

func New() {
	cfg := config.Load()

	dbManager, err := database.NewManager(*cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database manager: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: cfg.Server.UploadMaxSizeMB * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
				"code":  code,
			})
		},
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Requested-With",
	}))
	app.Use(recover.New())

	documentHandler := handlers.NewDocumentHandler(dbManager)
	uploadHandler := handlers.NewUploadHandler(dbManager, *cfg)
	docsHandler := handlers.NewDocsHandler()

	setupRoutes(app, documentHandler, uploadHandler, docsHandler)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Server starting on %s", address)
		if err := app.Listen(address); err != nil {
			log.Printf("Server failed to start: %v", err)
		}
	}()

	<-c
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := dbManager.Close(ctx); err != nil {
		log.Printf("Failed to close database connections: %v", err)
	}

	log.Println("Server exited properly")
}

func setupRoutes(app *fiber.App, documentHandler *handlers.DocumentHandler, uploadHandler *handlers.UploadHandler, docsHandler *handlers.DocsHandler) {
	app.Get("/", docsHandler.GetAPIInfo)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// Documentation routes
	app.Get("/docs", docsHandler.ServeScalarDocs)
	app.Get("/docs/openapi.yaml", docsHandler.ServeOpenAPISpec)

	api := app.Group("/api/v1")

	// Legacy upload endpoint (for backwards compatibility)
	api.Post("/upload-pdf", documentHandler.UploadPDF)
	
	// New chunked upload endpoints
	api.Post("/upload/initiate", uploadHandler.InitiateUpload)
	api.Post("/upload/chunk", uploadHandler.UploadChunk)
	api.Post("/upload/:sessionId/complete", uploadHandler.CompleteUpload)
	api.Delete("/upload/:sessionId/abort", uploadHandler.AbortUpload)
	api.Get("/upload/:sessionId/progress", uploadHandler.GetUploadProgress)
	
	// Document management endpoints
	api.Get("/documents", documentHandler.ListDocuments)
	api.Get("/documents/:id", documentHandler.GetDocument)
	api.Delete("/documents/:id", documentHandler.DeleteDocument)
	api.Post("/documents/:documentId/process-graph", documentHandler.ProcessDocumentWithGraphType)
	api.Get("/graphs/:id", documentHandler.GetGraphNetwork)
	api.Get("/graphs/:id/centroid", documentHandler.GetCentroid)
}
