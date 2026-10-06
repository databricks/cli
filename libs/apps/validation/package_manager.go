package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/apps/packagejson"
	"github.com/databricks/cli/libs/log"
)

const (
	packageManagerNpm  = "npm"
	packageManagerPnpm = "pnpm"
	packageManagerYarn = "yarn"
	packageManagerBun  = "bun"
	packageManagerKey  = "packageManager"
)

// DetectPackageManager prefers package.json's packageManager declaration over lockfiles
// in workDir. Unsupported declarations and lockfiles are logged and ignored. Without a
// supported declaration, npm lockfiles take precedence over pnpm, with npm as the default.
// The selected manager must be installed to run validation.
func DetectPackageManager(ctx context.Context, workDir string) (string, error) {
	lockfiles := []struct {
		name      string
		manager   string
		supported bool
	}{
		// https://docs.npmjs.com/cli/configuring-npm/package-lock-json
		// https://docs.npmjs.com/cli/configuring-npm/npm-shrinkwrap-json
		{"package-lock.json", packageManagerNpm, true},
		{"npm-shrinkwrap.json", packageManagerNpm, true},
		// https://pnpm.io/git
		{"pnpm-lock.yaml", packageManagerPnpm, true},
		// https://classic.yarnpkg.com/lang/en/docs/yarn-lock/
		{"yarn.lock", packageManagerYarn, false},
		// https://bun.sh/docs/install/lockfile
		{"bun.lock", packageManagerBun, false},
		{"bun.lockb", packageManagerBun, false},
	}

	// Lockfiles are listed npm-first, so the first supported one found wins and keeps
	// npm's precedence when there is no supported packageManager declaration.
	manager := readDeclaredPackageManager(ctx, workDir)
	chosen := packagejson.FileName
	for _, lockfile := range lockfiles {
		info, err := os.Stat(filepath.Join(workDir, lockfile.name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if !lockfile.supported {
			log.Warnf(ctx, "ignoring unsupported lockfile %s; apps validation uses %s or %s", lockfile.name, packageManagerNpm, packageManagerPnpm)
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
			log.Warnf(ctx, "found conflicting package managers; using %s from %s and ignoring %s", manager, chosen, lockfile.name)
		}
	}

	if manager == "" {
		return packageManagerNpm, nil
	}
	return manager, nil
}

// readDeclaredPackageManager leaves manifest validation to the selected package manager.
func readDeclaredPackageManager(ctx context.Context, workDir string) string {
	fields, err := packagejson.Read(workDir)
	if err != nil {
		// Preserve native manifest diagnostics instead of failing during detection.
		log.Debugf(ctx, "cannot read package manager metadata: %v", err)
		return ""
	}
	value, ok := fields[packageManagerKey]
	if !ok {
		return ""
	}
	var declaration string
	if err := json.Unmarshal(value, &declaration); err != nil {
		log.Warnf(ctx, "ignoring invalid %s in %s; expected a string", packageManagerKey, packagejson.FileName)
		return ""
	}
	if declaration == "" {
		return ""
	}

	// Only select the tool; its version policy remains native:
	// https://nodejs.org/api/packages.html#packagemanager.
	manager, _, _ := strings.Cut(declaration, "@")
	switch manager {
	case packageManagerNpm, packageManagerPnpm:
		return manager
	default:
		log.Warnf(ctx, "ignoring unsupported %s %q in %s; apps validation uses %s or %s", packageManagerKey, declaration, packagejson.FileName, packageManagerNpm, packageManagerPnpm)
		return ""
	}
}
