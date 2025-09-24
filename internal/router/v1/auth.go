package v1

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/identity"
)

func RegisterAuthRoutes(
	router fiber.Router,
	handler *identity.IdentityHandler,
	middleware *identity.AuthMiddleware,
) {
	auth := router.Group("/auth")

	auth.Post("/login", handler.Login)
	auth.Post("/register", handler.Register)

	auth.Post("/verify-email", handler.VerifyEmail)
	auth.Post("/resend-verification", handler.ResendEmailVerification)

	auth.Post("/forgot-password", handler.ForgotPassword)
	auth.Post("/reset-password", handler.ResetPassword)

	auth.Post("/validate", handler.ValidateToken)
	auth.Post("/refresh", handler.RefreshToken)

	auth.Post("/logout", middleware.RequireAuth(), handler.Logout)
	auth.Post("/change-password", middleware.RequireAuth(), handler.ChangePassword)
}