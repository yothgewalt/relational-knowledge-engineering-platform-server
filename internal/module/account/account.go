package account

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
)

func NewService(
	mongoService *mongo.MongoService,
	jwtService *jwt.JWTService,
	resendService resend.ResendService,
	fromEmail string,
) AccountService {
	return NewAccountService(mongoService, nil, jwtService, resendService, fromEmail)
}


func NewHandler(service AccountService) *AccountHandler {
	return NewAccountHandler(service)
}

func NewMiddleware(service AccountService) *AccountMiddleware {
	return NewAccountMiddleware(service)
}

type AccountModule struct {
	container.BaseModule
	fromEmail          string
	useCacheForOTP     bool
	useCacheForSession bool
}

func NewAccountModule(fromEmail string) *AccountModule {
	base := container.NewBaseModule(
		"account",
		"1.0.0",
		"Account management and authentication module",
		[]string{},
	)

	return &AccountModule{
		BaseModule:         base,
		fromEmail:          fromEmail,
		useCacheForOTP:     true,
		useCacheForSession: true,
	}
}

func (m *AccountModule) WithCacheConfig(useCacheForOTP, useCacheForSession bool) *AccountModule {
	m.useCacheForOTP = useCacheForOTP
	m.useCacheForSession = useCacheForSession
	return m
}

func (m *AccountModule) RegisterServices(registry *container.ServiceRegistry) error {
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

	var accountService AccountService

	cacheService := registry.GetRedis()
	accountService = NewAccountService(mongoService, cacheService, jwtService, resendService, m.fromEmail)

	if err := registry.RegisterService("account", accountService); err != nil {
		return err
	}

	return nil
}

func (m *AccountModule) RegisterRoutes(router fiber.Router, registry *container.ServiceRegistry) error {
	accountServiceInterface, err := registry.GetService("account")
	if err != nil {
		return err
	}

	accountService := accountServiceInterface.(AccountService)
	handler := NewHandler(accountService)
	middleware := NewMiddleware(accountService)

	accounts := router.Group("/accounts")

	accounts.Post("/login", handler.Login)
	accounts.Post("/register", handler.Register)
	accounts.Post("/logout", middleware.RequireAuth(), handler.Logout)
	accounts.Post("/refresh", handler.RefreshToken)
	accounts.Post("/validate", handler.ValidateToken)
	accounts.Post("/forgot-password", handler.ForgotPassword)
	accounts.Post("/reset-password", handler.ResetPassword)
	accounts.Post("/verify-email", handler.VerifyEmail)
	accounts.Post("/resend-verification", handler.ResendEmailVerification)
	accounts.Post("/change-password", middleware.RequireAuth(), handler.ChangePassword)

	accounts.Get("/", middleware.RequireAuth(), handler.ListAccounts)
	accounts.Post("/", handler.CreateAccount)
	accounts.Get("/me", middleware.RequireAuth(), handler.GetMe)

	accounts.Get("/email", middleware.OptionalAuth(), handler.GetAccountByEmail)
	accounts.Get("/username", middleware.OptionalAuth(), handler.GetAccountByUsername)

	accounts.Get("/:id", middleware.RequireAuth(), middleware.ValidateAccountOwnership(), handler.GetAccount)
	accounts.Put("/:id", middleware.RequireAuth(), middleware.ValidateAccountOwnership(), handler.UpdateAccount)
	accounts.Delete("/:id", middleware.RequireAuth(), middleware.ValidateAccountOwnership(), handler.DeleteAccount)

	return nil
}

func (m *AccountModule) RegisterMiddleware(registry *container.ServiceRegistry) error {
	return nil
}
