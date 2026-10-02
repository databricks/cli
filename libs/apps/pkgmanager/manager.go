package pkgmanager

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

// Pinned package manager versions written to package.json's "packageManager" field.
// These versions are the authoritative minimum versions required by the template.
// NOTE: pins are "coupled to the version threshold" — Phase 5 will add that threshold constant nearby; keep them co-located.
const (
	pnpmPin = "pnpm@11.0.8"

	// npmPin is the pinned npm version written to package.json's "packageManager".
	// TODO(VERIFY): confirm the canonical npm pin with the AppKit template owners; placeholder for now.
	npmPin = "npm@10.9.2"

	// pnpmDualPMThreshold is the first AppKit template version that supports dual package manager
	// (pnpm and npm). Template versions below this threshold only support npm.
	pnpmDualPMThreshold = "0.82.0"
)

// Manager represents a package manager with its properties.
type Manager struct {
	Name                string   // e.g., "npm", "pnpm"
	InstallCommand      string   // e.g., "npm ci", "pnpm install --frozen-lockfile"
	InstallArgs         []string // exec args after the binary name (e.g., ["ci", "--no-audit", ...])
	LockfileName        string   // e.g., "package-lock.json", "pnpm-lock.yaml"
	WorkspaceConfigName string   // pnpm-workspace.yaml for pnpm, unused for npm
	Pin                 string   // pinned version written to package.json's "packageManager", format "name@version"
}

// managers is the internal registry of supported package managers.
var managers = map[string]Manager{
	"pnpm": {
		Name:                "pnpm",
		InstallCommand:      "pnpm install --frozen-lockfile",
		InstallArgs:         []string{"install", "--frozen-lockfile"},
		LockfileName:        "pnpm-lock.yaml",
		WorkspaceConfigName: "pnpm-workspace.yaml",
		Pin:                 pnpmPin,
	},
	"npm": {
		Name:                "npm",
		InstallCommand:      "npm ci",
		InstallArgs:         []string{"ci", "--no-audit", "--no-fund", "--prefer-offline"},
		LockfileName:        "package-lock.json",
		WorkspaceConfigName: "",
		Pin:                 npmPin,
	},
}

// Resolve returns the Manager descriptor for the given package manager name.
// An empty name selects the default manager (pnpm).
// Returns an error if the name is unknown.
func Resolve(name string) (Manager, error) {
	if name == "" {
		return Default(), nil
	}
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

// EffectiveManager applies version-based constraints to determine the actual package manager to use.
// For pnpm requests against template versions below the dual-PM support threshold, it downgrades to npm.
// For npm requests, no downgrade occurs (npm is always supported).
// Returns the effective Manager and a downgraded bool (true only when a pnpm request becomes npm).
// Non-concrete versions (main, latest, empty, or branch names) are treated as >= threshold (no downgrade).
func EffectiveManager(selected Manager, resolvedVersion string) (Manager, bool) {
	// If the selected manager is already npm, no downgrade possible.
	if selected.Name != "pnpm" {
		return selected, false
	}

	// Non-concrete versions (main, latest, empty, or branch names) are treated as supporting dual-PM.
	// Only concrete semantic versions are subject to the threshold check.
	if !isSemverVersion(resolvedVersion) {
		return selected, false
	}

	// Extract the semantic version from template-v prefix if present.
	version := resolvedVersion
	const prefix = "template-v"
	version = strings.TrimPrefix(version, prefix)
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Compare the version against the threshold.
	// If version < threshold, downgrade to npm; otherwise honor the selection.
	threshold := "v" + pnpmDualPMThreshold
	if semver.Compare(version, threshold) < 0 {
		return managers["npm"], true
	}

	return selected, false
}

// isSemverVersion checks if a version string appears to be a semantic version.
// Returns false for non-concrete refs like "main", "latest", empty, or branch names.
func isSemverVersion(version string) bool {
	if version == "" || version == "main" {
		return false
	}

	// Strip template-v prefix if present to check the core version.
	const prefix = "template-v"
	core := strings.TrimPrefix(version, prefix)

	// Check if it looks like a semver: starts with v or a digit, and contains dots.
	// This rejects branch names like "feature-x" or "release-0.24" while accepting "0.25.0" and "v0.25.0".
	if (core != "" && (core[0] == 'v' || (core[0] >= '0' && core[0] <= '9'))) &&
		strings.Contains(core, ".") {
		return true
	}

	return false
}

// ValidateTemplate rejects templates with a lockfile only for another supported manager.
// Templates without lockfiles can still be scaffolded with --skip-install.
func (m Manager) ValidateTemplate(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, m.LockfileName)); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("check %s: %w", m.LockfileName, err)
	}
	for _, other := range managers {
		if other.Name == m.Name {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, other.LockfileName)); err == nil {
			return fmt.Errorf("template has %s but no %s required by %q; use --package-manager %s or add %s to the template", other.LockfileName, m.LockfileName, m.InstallCommand, other.Name, m.LockfileName)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("check %s: %w", other.LockfileName, err)
		}
	}
	return nil
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

// Rewrite mutates a decoded package.json (map[string]any) with package-manager-specific
// transformations: sets "packageManager" to the pinned version and normalizes scripts.
// Script invocations, including those in command chains, use explicit "run" form.
// Native package-manager operations and non-string script values are preserved.
// Rewrite modifies the map in place and returns it for convenience.
func Rewrite(pkg map[string]any, m Manager) map[string]any {
	// Set packageManager to the pinned version from the capability map.
	pkg["packageManager"] = m.Pin

	// Normalize script invocations without changing native package-manager operations.
	scripts, ok := pkg["scripts"].(map[string]any)
	if !ok {
		// No scripts or not a map — nothing to normalize.
		return pkg
	}

	for key, val := range scripts {
		scriptStr, ok := val.(string)
		if !ok {
			// Skip non-string script values (they may be arrays or objects in some esoteric setups).
			continue
		}

		scripts[key] = rewriteScript(scriptStr, m, scripts)
	}

	return pkg
}
