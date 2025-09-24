package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/identity"
)

func AuthMiddleware(identityService identity.IdentityService) *identity.AuthMiddleware {
	return identity.NewMiddleware(identityService)
}

func RequireAuth(identityService identity.IdentityService) fiber.Handler {
	authMiddleware := AuthMiddleware(identityService)
	return authMiddleware.RequireAuth()
}

func OptionalAuth(identityService identity.IdentityService) fiber.Handler {
	authMiddleware := AuthMiddleware(identityService)
	return authMiddleware.OptionalAuth()
}