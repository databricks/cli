package pkgmanager

import (
	"fmt"
	"slices"
	"strings"
)

// Manager represents a package manager with its properties.
type Manager struct {
	Name                 string // e.g., "npm", "pnpm"
	InstallCommand       string // e.g., "npm ci", "pnpm install --frozen-lockfile"
	LockfileName         string // e.g., "package-lock.json", "pnpm-lock.yaml"
	WorkspaceConfigName  string // pnpm-workspace.yaml for pnpm, unused for npm
}

// managers is the internal registry of supported package managers.
var managers = map[string]Manager{
	"pnpm": {
		Name:                "pnpm",
		InstallCommand:      "pnpm install --frozen-lockfile",
		LockfileName:        "pnpm-lock.yaml",
		WorkspaceConfigName: "pnpm-workspace.yaml",
	},
	"npm": {
		Name:                "npm",
		InstallCommand:      "npm ci",
		LockfileName:        "package-lock.json",
		WorkspaceConfigName: "npm",
	},
}

// Resolve returns the Manager descriptor for the given package manager name.
// Returns an error if the name is unknown.
func Resolve(name string) (Manager, error) {
	if m, ok := managers[name]; ok {
		return m, nil
	}
	// Build the list of allowed values from the managers map for a deterministic error message.
	allowed := make([]string, 0, len(managers))
	for k := range managers {
		allowed = append(allowed, k)
	}
	slices.Sort(allowed)
	return Manager{}, fmt.Errorf("unknown package manager %q (allowed: %s)", name, strings.Join(allowed, ", "))
}

// Default returns the default package manager (pnpm).
func Default() Manager {
	return managers["pnpm"]
}
