package pkgmanager

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// Pinned package manager versions written to package.json's "packageManager" field.
// These versions are the authoritative minimum versions required by the template.
// NOTE: pins are "coupled to the version threshold" — Phase 5 will add that threshold constant nearby; keep them co-located.
const (
	pnpmPin = "pnpm@11.0.8"

	// npmPin is the pinned npm version written to package.json's "packageManager".
	// TODO(VERIFY): confirm the canonical npm pin with the AppKit template owners; placeholder for now.
	npmPin = "npm@10.9.2"
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

// Rewrite mutates a decoded package.json (map[string]any) with package-manager-specific
// transformations: sets "packageManager" to the pinned version and normalizes scripts.
// Scripts whose leading token is a package manager name are rewritten to explicit
// "run" form (e.g., "npm build" becomes "pnpm run build" if pnpm is selected).
// Non-string script values and non-PM-leading scripts are left unchanged.
// Rewrite modifies the map in place and returns it for convenience.
func Rewrite(pkg map[string]any, m Manager) map[string]any {
	// Set packageManager to the pinned version from the capability map.
	pkg["packageManager"] = m.Pin

	// Normalize scripts: rewrite PM-leading commands to explicit "run" form.
	scripts, ok := pkg["scripts"].(map[string]any)
	if !ok {
		// No scripts or not a map — nothing to normalize.
		return pkg
	}

	// Known package manager names that can appear as leading tokens in scripts.
	pmNames := map[string]bool{
		"npm":  true,
		"pnpm": true,
		"yarn": true,
		"bun":  true,
	}

	for key, val := range scripts {
		scriptStr, ok := val.(string)
		if !ok {
			// Skip non-string script values (they may be arrays or objects in some esoteric setups).
			continue
		}

		// Extract the leading command token.
		leadingToken := extractLeadingToken(scriptStr)
		if leadingToken == "" || !pmNames[leadingToken] {
			// Not a PM command — leave unchanged.
			continue
		}

		// Rewrite: <pm> <subcommand> → <selected-pm> run <subcommand>
		// Strip the PM token and any run keyword that follows.
		rest := scriptStr[len(leadingToken):]
		rest = strings.TrimSpace(rest)

		// If rest already starts with "run ", remove it to avoid "run run".
		if strings.HasPrefix(rest, "run ") {
			rest = rest[4:]
			rest = strings.TrimSpace(rest)
		}

		// Construct the new script: "<selected-pm> run <rest>"
		newScript := m.Name + " run " + rest
		scripts[key] = newScript
	}

	return pkg
}

// extractLeadingToken extracts the first whitespace-delimited token from a string.
// Returns empty string if the input is empty or only whitespace.
func extractLeadingToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Find the first whitespace character.
	for i, r := range s {
		if unicode.IsSpace(r) {
			return s[:i]
		}
	}
	// No whitespace found — the entire string is the token.
	return s
}
