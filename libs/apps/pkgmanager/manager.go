package pkgmanager

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	// pnpmPin is the version required by the AppKit template.
	pnpmPin = "pnpm@11.0.8"

	// pnpmDualPMThreshold is the first AppKit template version that supports dual package manager
	// (pnpm and npm). Template versions below this threshold only support npm.
	pnpmDualPMThreshold = "0.82.0"
)

// Manager represents a package manager with its properties.
type Manager struct {
	Name                string   // e.g., "npm", "pnpm"
	InstallCommand      string   // e.g., "npm ci", "pnpm install --frozen-lockfile"
	InstallArgs         []string // exec args after the binary name (e.g., ["ci", "--no-audit", ...])
	LockfileNames       []string // In precedence order; npm prefers npm-shrinkwrap.json over package-lock.json.
	WorkspaceConfigName string   // pnpm-workspace.yaml for pnpm, unused for npm
	Pin                 string   // "name@version"; npm is empty until ResolvePin is called for an install.
}

// managers is the internal registry of supported package managers.
var managers = map[string]Manager{
	"pnpm": {
		Name:                "pnpm",
		InstallCommand:      "pnpm install --frozen-lockfile",
		InstallArgs:         []string{"install", "--frozen-lockfile"},
		LockfileNames:       []string{"pnpm-lock.yaml"},
		WorkspaceConfigName: "pnpm-workspace.yaml",
		Pin:                 pnpmPin,
	},
	"npm": {
		Name:                "npm",
		InstallCommand:      "npm ci",
		InstallArgs:         []string{"ci", "--no-audit", "--no-fund", "--prefer-offline"},
		LockfileNames:       []string{"npm-shrinkwrap.json", "package-lock.json"},
		WorkspaceConfigName: "",
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

// ResolvePin records the installed npm version for Node.js templates.
// Call only when installing; scaffolding with --skip-install must not require npm.
func (m Manager) ResolvePin(ctx context.Context, templateDir string) (Manager, error) {
	if m.Name != "npm" {
		return m, nil
	}
	for _, name := range []string{"package.json", "package.json.tmpl"} {
		if _, err := os.Stat(filepath.Join(templateDir, name)); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return m, fmt.Errorf("check %s: %w", name, err)
		}

		cmd := exec.CommandContext(ctx, m.Name, "--version")
		// Ignore inherited template/project pins when selecting the installed npm.
		// https://github.com/nodejs/corepack#environment-variables
		cmd.Env = append(os.Environ(), "COREPACK_ENABLE_PROJECT_SPEC=0")
		output, err := cmd.Output()
		if err != nil {
			return m, fmt.Errorf("detect npm version; ensure npm is installed and working, or use --skip-install: %w", err)
		}
		version := strings.TrimSpace(string(output))
		if !semver.IsValid("v" + version) {
			return m, fmt.Errorf("npm --version returned invalid version %q; ensure npm is working, or use --skip-install", version)
		}
		m.Pin = "npm@" + version
		return m, nil
	}
	return m, nil
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

// FindLockfile returns the lockfile used by the manager, or an empty name if none exists.
// npm's precedence is documented at https://docs.npmjs.com/cli/configuring-npm/npm-shrinkwrap-json.
func (m Manager) FindLockfile(dir string) (string, error) {
	for _, name := range m.LockfileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return name, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("check %s: %w", name, err)
		}
	}
	return "", nil
}

// ValidateTemplate rejects templates with a lockfile only for another supported manager.
// Templates without lockfiles can still be scaffolded with --skip-install.
func (m Manager) ValidateTemplate(dir string) error {
	name, err := m.FindLockfile(dir)
	if err != nil || name != "" {
		return err
	}
	for _, other := range managers {
		if other.Name == m.Name {
			continue
		}
		name, err := other.FindLockfile(dir)
		if err != nil {
			return err
		}
		if name != "" {
			lockfiles := strings.Join(m.LockfileNames, " or ")
			return fmt.Errorf("template has %s but no %s required by %q; use --package-manager %s or add %s to the template", name, lockfiles, m.InstallCommand, other.Name, lockfiles)
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
		for _, name := range mgr.LockfileNames {
			universe[name] = true
		}
		if mgr.WorkspaceConfigName != "" {
			universe[mgr.WorkspaceConfigName] = true
		}
	}

	// For the selected manager, identify which files to keep.
	keep := make(map[string]bool)
	for _, name := range m.LockfileNames {
		keep[name] = true
	}
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
// transformations: updates "packageManager" and normalizes scripts. Without a resolved
// pin, it preserves the template's pin only if it belongs to the selected manager.
// Script invocations, including those in command chains, use explicit "run" form.
// Native package-manager operations and non-string script values are preserved.
// Rewrite modifies the map in place and returns it for convenience.
func Rewrite(pkg map[string]any, m Manager) map[string]any {
	if m.Pin != "" {
		pkg["packageManager"] = m.Pin
	} else if pin, _ := pkg["packageManager"].(string); !strings.HasPrefix(pin, m.Name+"@") {
		delete(pkg, "packageManager")
	}

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
