package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// Detects if test is run from "debug test" feature in VS Code.
func isInDebug() bool {
	ex, _ := os.Executable()
	return strings.HasPrefix(path.Base(ex), "__debug_bin")
}

// Loads debug environment from ~/.databricks/debug-env.json.
func loadDebugEnvIfRunFromIDE(key string) error {
	if !isInDebug() {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find user home: %s", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".databricks/debug-env.json"))
	if err != nil {
		return fmt.Errorf("cannot load ~/.databricks/debug-env.json: %s", err)
	}
	var conf map[string]map[string]string
	err = json.Unmarshal(raw, &conf)
	if err != nil {
		return fmt.Errorf("cannot parse ~/.databricks/debug-env.json: %s", err)
	}
	vars, ok := conf[key]
	if !ok {
		return fmt.Errorf("%s is not configured in ~/.databricks/debug-env.json", key)
	}
	for k, v := range vars {
		os.Setenv(k, v)
	}
	return nil
}

// WorkspaceMain is the entry point for integration tests that run against a Databricks workspace.
// We use this for all integration tests defined in this subtree to ensure
// they are not inadvertently executed when calling `go test ./...`.
func WorkspaceMain(m *testing.M) {
	err := loadDebugEnvIfRunFromIDE("workspace")
	if err != nil {
		fmt.Printf("failed to load debug env: %s\n", err)
		return
	}

	value := os.Getenv("CLOUD_ENV")
	if value == "" {
		fmt.Println("CLOUD_ENV is not set, skipping integration tests")
		return
	}

	m.Run()
}
