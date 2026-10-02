package validation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

const (
	packageManagerNpm  = "npm"
	packageManagerPnpm = "pnpm"
	packageManagerYarn = "yarn"
	packageManagerBun  = "bun"
)

// DetectPackageManager selects a package manager from lockfiles in workDir. Unsupported
// lockfiles (yarn, bun) and lockfiles for a second manager are logged and ignored rather
// than failing detection. The selected manager must be installed to run validation.
// Projects without a supported lockfile use npm for compatibility with existing validation.
func DetectPackageManager(ctx context.Context, workDir string) (string, error) {
	lockfiles := []struct {
		name    string
		manager string
	}{
		// https://docs.npmjs.com/cli/configuring-npm/package-lock-json
		// https://docs.npmjs.com/cli/configuring-npm/npm-shrinkwrap-json
		{"package-lock.json", packageManagerNpm},
		{"npm-shrinkwrap.json", packageManagerNpm},
		// https://pnpm.io/git
		{"pnpm-lock.yaml", packageManagerPnpm},
		// https://classic.yarnpkg.com/lang/en/docs/yarn-lock/
		{"yarn.lock", packageManagerYarn},
		// https://bun.sh/docs/install/lockfile
		{"bun.lock", packageManagerBun},
		{"bun.lockb", packageManagerBun},
	}

	// Lockfiles are listed npm-first, so the first supported one found wins and keeps
	// npm's precedence for backward compatibility when managers conflict.
	var manager, chosen string
	for _, lockfile := range lockfiles {
		info, err := os.Stat(filepath.Join(workDir, lockfile.name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if lockfile.manager != packageManagerNpm && lockfile.manager != packageManagerPnpm {
			log.Warnf(ctx, "ignoring unsupported lockfile %s; apps validation uses npm or pnpm", lockfile.name)
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to inspect %s: %w", lockfile.name, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("lockfile %s must be a regular file", lockfile.name)
		}
		if manager == "" {
			manager, chosen = lockfile.manager, lockfile.name
			continue
		}
		if lockfile.manager != manager {
			log.Warnf(ctx, "found conflicting package manager lockfiles; using %s and ignoring %s", chosen, lockfile.name)
		}
	}

	if manager == "" {
		return packageManagerNpm, nil
	}
	return manager, nil
}
