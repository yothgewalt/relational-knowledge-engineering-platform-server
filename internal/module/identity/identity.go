package identity

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/module/account"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
)

func NewService(
	mongoService *mongo.MongoService,
	accountService account.AccountService,
	jwtService *jwt.JWTService,
	resendService resend.ResendService,
	fromEmail string,
) IdentityService {
	repository := NewIdentityRepository(mongoService)
	return NewIdentityService(repository, accountService, jwtService, resendService, fromEmail)
}

func NewHandler(service IdentityService) *IdentityHandler {
	return NewIdentityHandler(service)
}

func NewMiddleware(service IdentityService) *AuthMiddleware {
	return NewAuthMiddleware(service)
}

type IdentityModule struct {
	container.BaseModule
	fromEmail string
}

func NewIdentityModule(fromEmail string) *IdentityModule {
	base := container.NewBaseModule(
		"identity",
		"1.0.0",
		"Identity and authentication module",
		[]string{"account"},
	)
	
	return &IdentityModule{
		BaseModule: base,
		fromEmail:  fromEmail,
	}
}

func (m *IdentityModule) RegisterServices(registry *container.ServiceRegistry) error {
	mongoService := registry.GetMongo()
	if mongoService == nil {
		return container.ServiceNotFoundError{ServiceName: "mongo"}
	}

	jwtService := registry.GetJWT()
	if jwtService == nil {
		return container.ServiceNotFoundError{ServiceName: "jwt"}
	}

	resendService := registry.GetResend()
	if resendService == nil {
		return container.ServiceNotFoundError{ServiceName: "resend"}
	}

	accountServiceInterface, err := registry.GetService("account")
	if err != nil {
		return err
	}

	accountService, ok := accountServiceInterface.(account.AccountService)
	if !ok {
		return container.ServiceNotFoundError{ServiceName: "account with correct interface"}
	}

	identityService := NewService(mongoService, accountService, jwtService, resendService, m.fromEmail)
	
	if err := registry.RegisterService("identity", identityService); err != nil {
		return err
	}

	return nil
}

func (m *IdentityModule) RegisterRoutes(router fiber.Router, registry *container.ServiceRegistry) error {
	identityServiceInterface, err := registry.GetService("identity")
	if err != nil {
		return err
	}
	
	identityService := identityServiceInterface.(IdentityService)
	handler := NewHandler(identityService)
	middleware := NewMiddleware(identityService)

	auth := router.Group("/auth")

	auth.Post("/login", handler.Login)
	auth.Post("/register", handler.Register)
	auth.Post("/logout", middleware.RequireAuth(), handler.Logout)
	auth.Post("/refresh", handler.RefreshToken)
	
	auth.Post("/forgot-password", handler.ForgotPassword)
	auth.Post("/reset-password", handler.ResetPassword)
	auth.Post("/verify-email", handler.VerifyEmail)
	auth.Post("/resend-verification", handler.ResendEmailVerification)

	return nil
}

func (m *IdentityModule) RegisterMiddleware(registry *container.ServiceRegistry) error {
	return nil
}