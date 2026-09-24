package pkgmanager

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Manager represents a package manager with its properties.
type Manager struct {
	Name                string // e.g., "npm", "pnpm"
	InstallCommand      string // e.g., "npm ci", "pnpm install --frozen-lockfile"
	LockfileName        string // e.g., "package-lock.json", "pnpm-lock.yaml"
	WorkspaceConfigName string // pnpm-workspace.yaml for pnpm, unused for npm
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
		WorkspaceConfigName: "",
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

// Prune removes non-selected package manager artifacts from the given directory.
// It builds the universe of all PM-specific artifacts from the capability map,
// then removes all artifacts not belonging to the selected manager.
// The operation is idempotent: no error is returned if a target artifact is absent.
func (m Manager) Prune(dir string) error {
	// Build the universe: all PM-specific artifacts across all managers.
	universe := make(map[string]bool)
	for _, mgr := range managers {
		universe[mgr.LockfileName] = true
		if mgr.WorkspaceConfigName != "" {
			universe[mgr.WorkspaceConfigName] = true
		}
	}

	// For the selected manager, identify which files to keep.
	keep := make(map[string]bool)
	keep[m.LockfileName] = true
	if m.WorkspaceConfigName != "" {
		keep[m.WorkspaceConfigName] = true
	}

	// Remove all artifacts in the universe that are not in the keep set.
	for artifact := range universe {
		if !keep[artifact] {
			path := filepath.Join(dir, artifact)
			// Ignore not-exist errors for idempotency.
			if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("failed to remove %s: %w", artifact, err)
			}
		}
	}

	return nil
}
