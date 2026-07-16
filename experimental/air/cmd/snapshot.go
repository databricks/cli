package aircmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/env"
)

// resolveRootPath resolves a code_source snapshot root_path: expand environment
// variables and ~, strip a leading "project_root/" (meaning "relative to the YAML
// file"), and resolve the rest against the config's directory. It then confirms the
// path exists and is a directory.
func resolveRootPath(ctx context.Context, rawPath, configDir string) (string, error) {
	expanded := os.ExpandEnv(rawPath)
	if home, err := env.UserHomeDir(ctx); err == nil {
		if expanded == "~" {
			expanded = home
		} else if rest, ok := strings.CutPrefix(expanded, "~/"); ok {
			expanded = filepath.Join(home, rest)
		}
	}

	var resolved string
	switch {
	case strings.HasPrefix(expanded, "project_root/"):
		resolved = filepath.Join(configDir, strings.TrimPrefix(expanded, "project_root/"))
	case filepath.IsAbs(expanded):
		resolved = expanded
	default:
		resolved = filepath.Join(configDir, expanded)
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to resolve root_path %s: %w", resolved, err)
	}
	resolved = abs

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("root_path does not exist: %s", resolved)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root_path must be a directory: %s", resolved)
	}
	return resolved, nil
}
