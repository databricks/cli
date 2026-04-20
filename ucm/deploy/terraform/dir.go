package terraform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/ucm"
)

// cacheDirName is the per-root directory where ucm keeps its generated
// terraform config, plan artefacts, and (on M1+) a downloaded terraform
// binary. Mirrors bundle/deploy/terraform's `.databricks/bundle/<target>`
// layout but under a ucm-owned subtree so the two subcommands don't
// accidentally share state.
const cacheDirName = ".databricks/ucm"

// WorkingDir returns the absolute path of the terraform working directory
// for the currently selected target: `<root>/.databricks/ucm/<target>/terraform`.
// The directory is created on demand with 0o700 permissions.
//
// Errors if no target has been selected — the caller should have run
// SelectTarget (or SelectDefaultTarget) before reaching here.
func WorkingDir(u *ucm.Ucm) (string, error) {
	if u == nil {
		return "", errors.New("ucm is nil")
	}
	target := u.Config.Ucm.Target
	if target == "" {
		return "", errors.New("target not set; run SelectTarget before terraform operations")
	}
	dir := filepath.Join(u.RootPath, filepath.FromSlash(cacheDirName), target, "terraform")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create terraform working directory %s: %w", dir, err)
	}
	return dir, nil
}
