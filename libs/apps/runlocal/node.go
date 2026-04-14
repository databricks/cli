package runlocal

import (
	"context"
	"encoding/json"
	"os"

	"github.com/databricks/cli/libs/env"
)

const NODE_DEBUG_PORT = "9229"

type PackageJson struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

type NodeApp struct {
	config      *Config
	spec        *AppSpec
	packageJson *PackageJson
}

func NewNodeApp(config *Config, spec *AppSpec, packageJson *PackageJson) *NodeApp {
	if config.DebugPort == "" {
		config.DebugPort = NODE_DEBUG_PORT
	}

	return &NodeApp{
		config:      config,
		spec:        spec,
		packageJson: packageJson,
	}
}

func (n *NodeApp) PrepareEnvironment(ctx context.Context) error {
	// Install dependencies
	installArgs := []string{"npm", "install"}
	if err := runCommand(ctx, n.config.AppPath, installArgs); err != nil {
		return err
	}

	// Run build script if it exists
	if _, ok := n.packageJson.Scripts["build"]; ok {
		buildArgs := []string{"npm", "run", "build"}
		if err := runCommand(ctx, n.config.AppPath, buildArgs); err != nil {
			return err
		}
	}

	return nil
}

func (n *NodeApp) GetCommand(ctx context.Context, debug bool) ([]string, []string, error) {
	var cmdEnv []string
	if debug {
		cmdEnv = n.enableDebugging(ctx)
	}

	if n.spec.Command == nil {
		return []string{"npm", "run", "start"}, cmdEnv, nil
	}

	return n.spec.Command, cmdEnv, nil
}

// enableDebugging returns environment variables that enable Node.js debugging.
func (n *NodeApp) enableDebugging(ctx context.Context) []string {
	nodeOpts := env.Get(ctx, "NODE_OPTIONS")
	if nodeOpts != "" {
		nodeOpts += " "
	}
	nodeOpts += "--inspect=" + n.config.DebugPort
	return []string{"NODE_OPTIONS=" + nodeOpts}
}

func readPackageJson(packageJsonPath string) (*PackageJson, error) {
	packageJson, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return nil, err
	}

	var packageJsonObj PackageJson
	err = json.Unmarshal(packageJson, &packageJsonObj)
	if err != nil {
		return nil, err
	}
	return &packageJsonObj, nil
}
