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
	app.Get("/health", healthCheckHandler)

	docs := app.Group("/docs")
	docs.Get("/", docsHandler.ServeScalarDocs)
	docs.Get("/openapi.yaml", docsHandler.ServeOpenAPISpec)

	v1 := app.Group("/api/v1")

	documents := v1.Group("/documents")
	documents.Get("/", documentHandler.ListDocuments)
	documents.Get("/:id", documentHandler.GetDocument)
	documents.Delete("/:id", documentHandler.DeleteDocument)
	documents.Post("/:documentId/process-graph", documentHandler.ProcessDocumentWithGraphType)

	v1.Post("/upload-pdf", documentHandler.UploadPDF)

	upload := v1.Group("/upload")
	upload.Post("/initiate", uploadHandler.InitiateUpload)
	upload.Post("/chunk", uploadHandler.UploadChunk)
	upload.Post("/:sessionId/complete", uploadHandler.CompleteUpload)
	upload.Delete("/:sessionId/abort", uploadHandler.AbortUpload)
	upload.Get("/:sessionId/progress", uploadHandler.GetUploadProgress)

	graphs := v1.Group("/graphs")
	graphs.Get("/:id", documentHandler.GetGraphNetwork)
	graphs.Get("/:id/centroid", documentHandler.GetCentroid)
}

func healthCheckHandler(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}
