package apps

import (
	"context"
	"encoding/json"
	"os"
)

const NODE_DEBUG_PORT = "9229"

type PackageJson struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

type NodeApp struct {
	ctx         context.Context
	config      *Config
	spec        *AppSpec
	packageJson *PackageJson
}

func NewNodeApp(ctx context.Context, config *Config, spec *AppSpec, packageJson *PackageJson) *NodeApp {
	if config.DebugPort == "" {
		config.DebugPort = NODE_DEBUG_PORT
	}

	return &NodeApp{
		ctx:         ctx,
		config:      config,
		spec:        spec,
		packageJson: packageJson,
	}
}

func (n *NodeApp) PrepareEnvironment() error {
	// Install dependencies
	installArgs := []string{"npm", "install"}
	if err := runCommand(n.ctx, n.config.AppPath, installArgs); err != nil {
		return err
	}

	// Run build script if it exists
	if _, ok := n.packageJson.Scripts["build"]; ok {
		buildArgs := []string{"npm", "run", "build"}
		if err := runCommand(n.ctx, n.config.AppPath, buildArgs); err != nil {
			return err
		}
	}

	return nil
}

func (n *NodeApp) GetCommand(debug bool) ([]string, error) {
	if debug {
		n.enableDebugging()
	}

	if n.spec.Command == nil {
		return []string{"npm", "run", "start"}, nil
	}

	return n.spec.Command, nil
}

func (n *NodeApp) enableDebugging() {
	// Set NODE_OPTIONS environment variable to enable debugging
	// This will make Node.js listen for debugger connections on the debug port
	if os.Getenv("NODE_OPTIONS") == "" {
		os.Setenv("NODE_OPTIONS", "--inspect="+n.config.DebugPort)
	} else {
		// If NODE_OPTIONS already exists, append the inspect flag
		os.Setenv("NODE_OPTIONS", os.Getenv("NODE_OPTIONS")+" --inspect="+n.config.DebugPort)
	}
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
