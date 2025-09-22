package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/yokeTH/gofiber-scalar/scalar/v2"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName:      "Relational Knowledge Engineering Platform",
		ServerHeader: "Fiber",
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	scalarConfigV1 := scalar.Config{
		BasePath: "/api/v1",
		Path:     "docs",
		Title:    "Relational Knowledge Engineering Platform API Documentation",
	}
	app.Use(scalar.New(scalarConfigV1))
	app.Use(cors.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"message":   "Server is running",
			"timestamp": time.Now(),
		})
	})

	v1 := app.Group("/v1")
	v1.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "1.0.0",
			"message": "Welcome to API v1",
		})
	})
	v1.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "v1",
			"users": []fiber.Map{
				{"id": 1, "name": "John Doe", "email": "john@example.com"},
				{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
			},
		})
	})
	v1.Get("/knowledge", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "v1",
			"knowledge_base": []fiber.Map{
				{"id": 1, "title": "Machine Learning Basics", "category": "AI"},
				{"id": 2, "title": "Database Design Principles", "category": "Database"},
			},
		})
	})

	v2 := app.Group("/v2")
	v2.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version":  "2.0.0",
			"message":  "Welcome to API v2",
			"features": []string{"enhanced performance", "new endpoints", "improved responses"},
		})
	})
	v2.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "v2",
			"meta": fiber.Map{
				"total": 2,
				"page":  1,
				"limit": 10,
			},
			"data": []fiber.Map{
				{
					"id":    1,
					"name":  "John Doe",
					"email": "john@example.com",
					"profile": fiber.Map{
						"age":        30,
						"department": "Engineering",
					},
				},
				{
					"id":    2,
					"name":  "Jane Smith",
					"email": "jane@example.com",
					"profile": fiber.Map{
						"age":        28,
						"department": "Research",
					},
				},
			},
		})
	})
	v2.Get("/knowledge", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "v2",
			"meta": fiber.Map{
				"total":      2,
				"categories": []string{"AI", "Database"},
			},
			"data": []fiber.Map{
				{
					"id":       1,
					"title":    "Machine Learning Basics",
					"category": "AI",
					"metadata": fiber.Map{
						"difficulty":     "beginner",
						"estimated_time": "2 hours",
						"tags":           []string{"ml", "basics", "tutorial"},
					},
				},
				{
					"id":       2,
					"title":    "Database Design Principles",
					"category": "Database",
					"metadata": fiber.Map{
						"difficulty":     "intermediate",
						"estimated_time": "3 hours",
						"tags":           []string{"database", "design", "principles"},
					},
				},
			},
		})
	})

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}

		log.Printf("Server starting on port %s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
