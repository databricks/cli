package dockercredentials

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

// Configuration describes the installed helper and the Docker configuration it updated.
type Configuration struct {
	ConfigPath string
	Shim       ShimInstallResult
}

// Status reports Docker's credential helper selection for a registry.
type Status struct {
	Host       string `json:"host"`
	Configured bool   `json:"configured"`
}

// Configure installs the helper before selecting it in Docker's configuration.
func Configure(ctx context.Context, executable, registryHost string) (Configuration, error) {
	shim, err := InstallShim(executable)
	if err != nil {
		return Configuration{}, fmt.Errorf("install Docker credential helper: %w", err)
	}
	configPath, err := ConfigPath(ctx)
	if err != nil {
		return Configuration{}, err
	}
	if err := SetCredentialHelper(configPath, registryHost); err != nil {
		return Configuration{}, fmt.Errorf("update Docker config %s: %w", filepath.ToSlash(configPath), err)
	}
	return Configuration{ConfigPath: configPath, Shim: shim}, nil
}

// Inspect reads the helper selection without modifying Docker's configuration.
func Inspect(ctx context.Context, registryHost string) (Status, error) {
	configPath, err := ConfigPath(ctx)
	if err != nil {
		return Status{}, err
	}
	configured, err := CredentialHelperConfigured(configPath, registryHost)
	if err != nil {
		return Status{}, err
	}
	return Status{Host: registryHost, Configured: configured}, nil
}

// ConfigPath resolves Docker's configuration path using DOCKER_CONFIG or the user's home directory.
func ConfigPath(ctx context.Context) (string, error) {
	if dir := env.Get(ctx, "DOCKER_CONFIG"); dir != "" {
		return filepath.Join(dir, "config.json"), nil
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}
