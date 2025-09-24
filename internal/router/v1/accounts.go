package v1

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/account"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/identity"
)

func RegisterAccountRoutes(
	router fiber.Router,
	handler *account.AccountHandler,
	accountMiddleware *account.AccountMiddleware,
	authMiddleware *identity.AuthMiddleware,
) {
	accounts := router.Group("/accounts")

	accounts.Get("/", authMiddleware.RequireAuth(), handler.ListAccounts)
	accounts.Post("/", handler.CreateAccount)

	accounts.Get("/email", authMiddleware.OptionalAuth(), handler.GetAccountByEmail)
	accounts.Get("/username", authMiddleware.OptionalAuth(), handler.GetAccountByUsername)

	accounts.Get("/:id", authMiddleware.RequireAuth(), accountMiddleware.ValidateAccountOwnership(), handler.GetAccount)
	accounts.Put("/:id", authMiddleware.RequireAuth(), accountMiddleware.ValidateAccountOwnership(), handler.UpdateAccount)
	accounts.Delete("/:id", authMiddleware.RequireAuth(), accountMiddleware.ValidateAccountOwnership(), handler.DeleteAccount)
}