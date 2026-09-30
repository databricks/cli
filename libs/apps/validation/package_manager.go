package validation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	packageManagerNpm  = "npm"
	packageManagerPnpm = "pnpm"
	packageManagerYarn = "yarn"
	packageManagerBun  = "bun"
)

// DetectPackageManager selects a package manager from lockfiles in workDir.
// Projects without lockfiles use npm for compatibility with existing validation.
func DetectPackageManager(workDir string) (string, error) {
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

	var manager string
	var found []string
	var conflict bool
	for _, lockfile := range lockfiles {
		info, err := os.Stat(filepath.Join(workDir, lockfile.name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to inspect %s: %w", lockfile.name, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("lockfile %s must be a regular file", lockfile.name)
		}

		found = append(found, lockfile.name)
		if manager != "" && manager != lockfile.manager {
			conflict = true
		}
		manager = lockfile.manager
	}

	if conflict {
		return "", fmt.Errorf("conflicting package manager lockfiles: %s; keep lockfiles for only one package manager", strings.Join(found, ", "))
	}
	if manager == "" {
		return packageManagerNpm, nil
	}
	return manager, nil
}
