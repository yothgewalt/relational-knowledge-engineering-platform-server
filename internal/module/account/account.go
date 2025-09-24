package account

import (
	"github.com/gofiber/fiber/v2"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/container"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
)

func NewService(mongoService *mongo.MongoService) AccountService {
	return NewAccountService(mongoService)
}

func NewHandler(service AccountService) *AccountHandler {
	return NewAccountHandler(service)
}

func NewMiddleware(service AccountService) *AccountMiddleware {
	return NewAccountMiddleware(service)
}

type AccountModule struct {
	container.BaseModule
}

func NewAccountModule() *AccountModule {
	base := container.NewBaseModule(
		"account",
		"1.0.0",
		"Account management module",
		[]string{},
	)

	return &AccountModule{
		BaseModule: base,
	}
}

func (m *AccountModule) RegisterServices(registry *container.ServiceRegistry) error {
	mongoService := registry.GetMongo()
	if mongoService == nil {
		return container.ServiceNotFoundError{ServiceName: "mongo"}
	}

	accountService := NewService(mongoService)

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

	identityServiceInterface, err := registry.GetService("identity")
	if err != nil {
		return err
	}

	identityService, ok := identityServiceInterface.(interface {
		ValidateToken(ctx interface{}, token string) (interface{}, error)
	})
	if !ok {
		return container.ServiceNotFoundError{ServiceName: "identity with ValidateToken"}
	}

	authMiddleware := &AuthenticationMiddleware{identityService: identityService}

	accounts := router.Group("/accounts")

	accounts.Get("/", authMiddleware.RequireAuth(), handler.ListAccounts)
	accounts.Post("/", handler.CreateAccount)

	accounts.Get("/email", authMiddleware.OptionalAuth(), handler.GetAccountByEmail)
	accounts.Get("/username", authMiddleware.OptionalAuth(), handler.GetAccountByUsername)

	accounts.Get("/:id", authMiddleware.RequireAuth(), middleware.ValidateAccountOwnership(), handler.GetAccount)
	accounts.Put("/:id", authMiddleware.RequireAuth(), middleware.ValidateAccountOwnership(), handler.UpdateAccount)
	accounts.Delete("/:id", authMiddleware.RequireAuth(), middleware.ValidateAccountOwnership(), handler.DeleteAccount)

	return nil
}

func (m *AccountModule) RegisterMiddleware(registry *container.ServiceRegistry) error {
	return nil
}

type AuthenticationMiddleware struct {
	identityService interface {
		ValidateToken(ctx interface{}, token string) (interface{}, error)
	}
}

func (m *AuthenticationMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

func (m *AuthenticationMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
