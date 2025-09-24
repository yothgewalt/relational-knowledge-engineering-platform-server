package container

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/rs/zerolog"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/jwt"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/mongo"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/neo4j"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/redis"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/resend"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/package/vault"
)

type ServiceRegistry struct {
	logger zerolog.Logger
	mu     sync.RWMutex

	mongoService  *mongo.MongoService
	redisService  redis.RedisService
	neo4jService  neo4j.Neo4jService
	resendService resend.ResendService
	vaultService  vault.VaultService
	jwtService    *jwt.JWTService

	services        map[string]interface{}
	serviceTypes    map[string]reflect.Type
	dependencies    map[string][]string
	initialized     map[string]bool
	initializing    map[string]bool
}

type ServiceNotFoundError struct {
	ServiceName string
}

func (e ServiceNotFoundError) Error() string {
	return fmt.Sprintf("service '%s' not found in registry", e.ServiceName)
}

type CircularDependencyError struct {
	ServiceName string
	Chain       []string
}

func (e CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected for service '%s': %v", e.ServiceName, e.Chain)
}

func NewServiceRegistry(logger zerolog.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		logger:       logger,
		services:     make(map[string]interface{}),
		serviceTypes: make(map[string]reflect.Type),
		dependencies: make(map[string][]string),
		initialized:  make(map[string]bool),
		initializing: make(map[string]bool),
	}
}

func (r *ServiceRegistry) RegisterInfrastructure(
	mongoService *mongo.MongoService,
	redisService redis.RedisService,
	neo4jService neo4j.Neo4jService,
	resendService resend.ResendService,
	vaultService vault.VaultService,
	jwtService *jwt.JWTService,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.mongoService = mongoService
	r.redisService = redisService
	r.neo4jService = neo4jService
	r.resendService = resendService
	r.vaultService = vaultService
	r.jwtService = jwtService

	r.logger.Info().Msg("Infrastructure services registered in service registry")
}

func (r *ServiceRegistry) RegisterService(name string, service interface{}, dependencies ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service '%s' is already registered", name)
	}

	r.services[name] = service
	r.serviceTypes[name] = reflect.TypeOf(service)
	r.dependencies[name] = dependencies
	r.initialized[name] = true

	r.logger.Info().
		Str("service", name).
		Strs("dependencies", dependencies).
		Msg("Service registered in registry")

	return nil
}

func (r *ServiceRegistry) GetService(name string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	if !exists {
		return nil, ServiceNotFoundError{ServiceName: name}
	}

	return service, nil
}

func (r *ServiceRegistry) MustGetService(name string) interface{} {
	service, err := r.GetService(name)
	if err != nil {
		panic(fmt.Sprintf("Required service '%s' not found: %v", name, err))
	}
	return service
}

func (r *ServiceRegistry) GetServiceOfType(serviceType interface{}) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	targetType := reflect.TypeOf(serviceType)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	for _, service := range r.services {
		serviceType := reflect.TypeOf(service)
		if serviceType == targetType || (serviceType.Kind() == reflect.Ptr && serviceType.Elem() == targetType) {
			return service, nil
		}

		if serviceType.Kind() == reflect.Interface || targetType.Kind() == reflect.Interface {
			if serviceType.Implements(targetType) || reflect.TypeOf(service).Implements(targetType) {
				return service, nil
			}
		}
	}

	return nil, fmt.Errorf("no service found implementing type %v", targetType)
}

func (r *ServiceRegistry) GetMongo() *mongo.MongoService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mongoService
}

func (r *ServiceRegistry) GetRedis() redis.RedisService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.redisService
}

func (r *ServiceRegistry) GetNeo4j() neo4j.Neo4jService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.neo4jService
}

func (r *ServiceRegistry) GetResend() resend.ResendService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resendService
}

func (r *ServiceRegistry) GetVault() vault.VaultService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.vaultService
}

func (r *ServiceRegistry) GetJWT() *jwt.JWTService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.jwtService
}

func (r *ServiceRegistry) HasService(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.services[name]
	return exists
}

func (r *ServiceRegistry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}
	return services
}

func (r *ServiceRegistry) ValidateDependencies() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for serviceName := range r.services {
		if err := r.validateServiceDependencies(serviceName, []string{}); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceRegistry) validateServiceDependencies(serviceName string, chain []string) error {
	for _, visited := range chain {
		if visited == serviceName {
			return CircularDependencyError{
				ServiceName: serviceName,
				Chain:       append(chain, serviceName),
			}
		}
	}

	dependencies, exists := r.dependencies[serviceName]
	if !exists {
		return nil
	}

	newChain := append(chain, serviceName)
	for _, dep := range dependencies {
		if err := r.validateServiceDependencies(dep, newChain); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceRegistry) GetServiceInfo(name string) (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	if !exists {
		return nil, ServiceNotFoundError{ServiceName: name}
	}

	return map[string]interface{}{
		"name":         name,
		"type":         r.serviceTypes[name].String(),
		"dependencies": r.dependencies[name],
		"initialized":  r.initialized[name],
		"service":      service,
	}, nil
}

func (r *ServiceRegistry) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	for name := range r.services {
		if err := r.shutdownService(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown service '%s': %w", name, err))
		}
	}

	r.services = make(map[string]interface{})
	r.serviceTypes = make(map[string]reflect.Type)
	r.dependencies = make(map[string][]string)
	r.initialized = make(map[string]bool)
	r.initializing = make(map[string]bool)

	if len(errors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors: %v", len(errors), errors)
	}

	r.logger.Info().Msg("Service registry shutdown completed")
	return nil
}

func (r *ServiceRegistry) shutdownService(name string) error {
	service, exists := r.services[name]
	if !exists {
		return nil
	}

	if shutdownable, ok := service.(interface{ Shutdown() error }); ok {
		return shutdownable.Shutdown()
	}

	if closer, ok := service.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}