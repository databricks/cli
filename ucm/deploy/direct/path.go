package direct

import (
	"path/filepath"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
)

// StatePath returns the canonical on-disk location of the direct-engine
// state file for u's currently-selected target. Sits beside the terraform
// engine's own artefacts under `.databricks/ucm/<target>/`.
func StatePath(u *ucm.Ucm) string {
	return filepath.Join(u.RootPath, filepath.FromSlash(deploy.LocalCacheDir), u.Config.Ucm.Target, StateFileName)
}
