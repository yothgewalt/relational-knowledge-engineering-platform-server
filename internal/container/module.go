package container

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type ModuleInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Author       string   `json:"author"`
}

type Module interface {
	Info() ModuleInfo
	RegisterServices(registry *ServiceRegistry) error
	RegisterRoutes(router fiber.Router, registry *ServiceRegistry) error
	RegisterMiddleware(registry *ServiceRegistry) error
}

type BaseModule struct {
	info ModuleInfo
}

func NewBaseModule(name, version, description string, dependencies []string) BaseModule {
	return BaseModule{
		info: ModuleInfo{
			Name:         name,
			Version:      version,
			Description:  description,
			Dependencies: dependencies,
		},
	}
}

func (m BaseModule) Info() ModuleInfo {
	return m.info
}

func (m BaseModule) RegisterServices(registry *ServiceRegistry) error {
	return nil
}

func (m BaseModule) RegisterRoutes(router fiber.Router, registry *ServiceRegistry) error {
	return nil
}

func (m BaseModule) RegisterMiddleware(registry *ServiceRegistry) error {
	return nil
}

type ModuleManager struct {
	modules  []Module
	registry *ServiceRegistry
	logger   zerolog.Logger
}

func NewModuleManager(registry *ServiceRegistry, logger zerolog.Logger) *ModuleManager {
	return &ModuleManager{
		modules:  make([]Module, 0),
		registry: registry,
		logger:   logger,
	}
}

func (mm *ModuleManager) RegisterModule(module Module) error {
	info := module.Info()

	if err := mm.validateDependencies(info.Dependencies); err != nil {
		return err
	}

	mm.modules = append(mm.modules, module)

	mm.logger.Info().Str("module", info.Name).Str("version", info.Version).Msg("Module registered")

	return nil
}

func (mm *ModuleManager) InitializeServices() error {
	for _, module := range mm.modules {
		info := module.Info()

		if err := module.RegisterServices(mm.registry); err != nil {
			return err
		}

		mm.logger.Info().Str("module", info.Name).Msg("Module services initialized")
	}

	if err := mm.registry.ValidateDependencies(); err != nil {
		return err
	}

	return nil
}

func (mm *ModuleManager) InitializeMiddleware() error {
	for _, module := range mm.modules {
		info := module.Info()

		if err := module.RegisterMiddleware(mm.registry); err != nil {
			return err
		}

		mm.logger.Info().Str("module", info.Name).Msg("Module middleware initialized")
	}

	return nil
}

func (mm *ModuleManager) InitializeRoutes(router fiber.Router) error {
	for _, module := range mm.modules {
		info := module.Info()

		if info.Name == "docs" {
			continue
		}

		if err := module.RegisterRoutes(router, mm.registry); err != nil {
			return err
		}

		mm.logger.Info().Str("module", info.Name).Msg("Module routes initialized")
	}

	return nil
}

func (mm *ModuleManager) InitializeDocsRoutes(router fiber.Router) error {
	for _, module := range mm.modules {
		info := module.Info()

		if info.Name == "docs" {
			if err := module.RegisterRoutes(router, mm.registry); err != nil {
				return err
			}

			mm.logger.Info().Str("module", info.Name).Msg("Docs module routes initialized on main router")
			mm.logger.Info().Msg("API documentation available at: http://localhost:3000/docs")
			break
		}
	}

	return nil
}

func (mm *ModuleManager) GetModules() []Module {
	return mm.modules
}

func (mm *ModuleManager) GetModuleInfo() []ModuleInfo {
	info := make([]ModuleInfo, len(mm.modules))
	for i, module := range mm.modules {
		info[i] = module.Info()
	}
	return info
}

func (mm *ModuleManager) validateDependencies(dependencies []string) error {
	for _, dep := range dependencies {
		found := false
		for _, module := range mm.modules {
			if module.Info().Name == dep {
				found = true
				break
			}
		}

		if !found {
			return ServiceNotFoundError{ServiceName: dep}
		}
	}

	return nil
}
