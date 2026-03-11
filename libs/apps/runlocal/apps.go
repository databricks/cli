package runlocal

import (
	"context"
	"os"
	"path/filepath"
)

type App interface {
	PrepareEnvironment(ctx context.Context) error
	GetCommand(ctx context.Context, debug bool) (cmd, env []string, err error)
}

func NewApp(config *Config, spec *AppSpec) (App, error) {
	// Check if the app is a Node.js app by checking if there is a package.json file in the root of the app
	packageJsonPath := filepath.Join(config.AppPath, "package.json")
	_, err := os.Stat(packageJsonPath)
	if err == nil {
		// Read the package.json file
		packageJson, err := readPackageJson(packageJsonPath)
		if err != nil {
			return nil, err
		}
		return NewNodeApp(config, spec, packageJson), nil
	}

	return NewPythonApp(config, spec), nil
}
