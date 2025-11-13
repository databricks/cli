package sandbox

import (
	"fmt"
	"time"
)

// Type represents the type of sandbox implementation.
type Type string

const (
	// TypeLocal uses the local filesystem for sandbox operations.
	TypeLocal Type = "local"

	// TypeDagger uses Dagger containers for sandbox operations.
	TypeDagger Type = "dagger"
)

// Config holds the configuration for creating a sandbox.
type Config struct {
	BaseDir string
	Timeout time.Duration
}

// Option is a functional option for configuring sandbox creation.
type Option func(*Config)

// WithBaseDir sets the base directory for the sandbox.
// This is required for local sandboxes.
func WithBaseDir(dir string) Option {
	return func(c *Config) {
		c.BaseDir = dir
	}
}

// WithTimeout sets the default timeout for sandbox operations.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

// NewConfig creates a config from options.
func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		Timeout: 5 * time.Minute, // Default timeout
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// FactoryFunc is a function that creates a sandbox from configuration.
type FactoryFunc func(*Config) (Sandbox, error)

var factories = make(map[Type]FactoryFunc)

// Register registers a sandbox factory for a specific type.
func Register(typ Type, factory FactoryFunc) {
	factories[typ] = factory
}

// New creates a new sandbox of the specified type with the given options.
func New(typ Type, opts ...Option) (Sandbox, error) {
	cfg := NewConfig(opts...)

	factory, ok := factories[typ]
	if !ok {
		return nil, fmt.Errorf("unknown sandbox type: %s", typ)
	}

	return factory(cfg)
}
