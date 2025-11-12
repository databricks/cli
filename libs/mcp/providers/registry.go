// Package providers implements a registry pattern for automatic provider discovery and initialization.
package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/session"
)

// Provider is the interface that all MCP providers must implement.
// Providers are responsible for registering their tools with the MCP server.
// Note: RegisterTools is not included in the interface due to type constraints,
// but providers should implement it with the appropriate server type.
type Provider interface {
	// Name returns the unique name of the provider.
	Name() string
}

// ProviderFactory is a function that creates a new provider instance.
// It receives configuration, session, and logger instances.
type ProviderFactory func(cfg *mcp.Config, sess *session.Session, ctx context.Context) (Provider, error)

// Registry manages provider registration and creation.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
	config    map[string]ProviderConfig
}

// ProviderConfig holds configuration for conditional provider registration.
type ProviderConfig struct {
	// Always indicates the provider should always be registered.
	Always bool
	// EnabledWhen is a function that determines if the provider should be enabled.
	// If nil and Always is false, the provider won't be registered.
	EnabledWhen func(*mcp.Config) bool
}

var (
	globalRegistry *Registry
	once           sync.Once
)

// GetRegistry returns the global provider registry singleton.
func GetRegistry() *Registry {
	once.Do(func() {
		globalRegistry = &Registry{
			factories: make(map[string]ProviderFactory),
			config:    make(map[string]ProviderConfig),
		}
	})
	return globalRegistry
}

// Register registers a provider factory with the global registry.
// This is typically called from provider package init() functions.
func Register(name string, factory ProviderFactory, cfg ProviderConfig) {
	GetRegistry().RegisterProvider(name, factory, cfg)
}

// RegisterProvider registers a provider factory with this registry.
func (r *Registry) RegisterProvider(name string, factory ProviderFactory, cfg ProviderConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		panic(fmt.Sprintf("provider %q already registered", name))
	}

	r.factories[name] = factory
	r.config[name] = cfg
}

// Create creates a provider instance by name.
func (r *Registry) Create(name string, cfg *mcp.Config, sess *session.Session, ctx context.Context) (Provider, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %q not registered", name)
	}

	return factory(cfg, sess, ctx)
}

// CreateAll creates all registered providers that are enabled according to their configuration.
func (r *Registry) CreateAll(cfg *mcp.Config, sess *session.Session, ctx context.Context) ([]Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []Provider

	for name, factory := range r.factories {
		providerCfg := r.config[name]

		shouldEnable := providerCfg.Always
		if !shouldEnable && providerCfg.EnabledWhen != nil {
			shouldEnable = providerCfg.EnabledWhen(cfg)
		}

		if !shouldEnable {
			continue
		}

		provider, err := factory(cfg, sess, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %q: %w", name, err)
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// List returns the names of all registered providers.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// CreateAll is a convenience function that uses the global registry.
func CreateAll(cfg *mcp.Config, sess *session.Session, ctx context.Context) ([]Provider, error) {
	return GetRegistry().CreateAll(cfg, sess, ctx)
}
