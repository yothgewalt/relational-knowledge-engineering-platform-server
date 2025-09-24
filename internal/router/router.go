package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/account"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/identity"
	v1 "github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/router/v1"
)

type RouterConfig struct {
	IdentityService identity.IdentityService
	AccountService  account.AccountService
}

func Setup(config RouterConfig) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "Relational Knowledge Engineering Platform",
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Relational Knowledge Engineering Platform API",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"timestamp": fiber.Map{
				"unix": time.Now().Unix(),
			},
		})
	})

	api := app.Group("/api")
	apiV1 := api.Group("/v1")

	identityHandler := identity.NewHandler(config.IdentityService)
	identityMiddleware := identity.NewMiddleware(config.IdentityService)

	accountHandler := account.NewHandler(config.AccountService)
	accountMiddleware := account.NewMiddleware(config.AccountService)

	v1.RegisterAuthRoutes(apiV1, identityHandler, identityMiddleware)
	v1.RegisterAccountRoutes(apiV1, accountHandler, accountMiddleware, identityMiddleware)

	app.Use("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Not Found",
			"message": "The requested resource was not found",
			"path":    c.Path(),
		})
	})

	return app
}