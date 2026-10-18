package runlocal

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/apps/validation"
)

type App interface {
	PrepareEnvironment(ctx context.Context) error
	GetCommand(ctx context.Context, debug bool) (cmd, env []string, err error)
}

func NewApp(ctx context.Context, config *Config, spec *AppSpec) (App, error) {
	// Check if the app is a Node.js app by checking if there is a package.json file in the root of the app
	packageJsonPath := filepath.Join(config.AppPath, "package.json")
	_, err := os.Stat(packageJsonPath)
	if err == nil {
		// Read the package.json file
		packageJson, err := readPackageJson(packageJsonPath)
		if err != nil {
			return nil, err
		}
		// Use the same package manager as apps validate and apps deploy.
		packageManager, err := validation.DetectPackageManager(ctx, config.AppPath)
		if err != nil {
			return nil, err
		}
		return NewNodeApp(config, spec, packageJson, packageManager), nil
	}

	return NewPythonApp(config, spec), nil
}
