package snapshot

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

const snapshotPathStateFile = "snapshot_path"

type (
	saveState struct{}
	loadState struct{}
)

// SaveState writes the snapshot path to the local deployment state directory
// so it can be recovered during destroy without reading metadata.json.
func SaveState() bundle.Mutator {
	return &saveState{}
}

// LoadState reads the snapshot path from the local deployment state directory
// and sets workspace.snapshot_path. Missing state is treated as a no-op so
// destroy can proceed against bundles deployed before this feature was added.
func LoadState() bundle.Mutator {
	return &loadState{}
}

func (s *saveState) Name() string { return "snapshot.SaveState" }
func (s *loadState) Name() string { return "snapshot.LoadState" }

func (s *saveState) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Workspace.SnapshotPath == "" {
		return nil
	}

	dir, err := b.LocalStateDir(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	p := filepath.Join(dir, snapshotPathStateFile)
	return diag.FromErr(os.WriteFile(p, []byte(b.Config.Workspace.SnapshotPath), 0o600))
}

func (s *loadState) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	dir := b.GetLocalStateDir(ctx)
	data, err := os.ReadFile(filepath.Join(dir, snapshotPathStateFile))

	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	snapshotPath := strings.TrimSpace(string(data))
	b.Config.Workspace.SnapshotPath = snapshotPath

	// Restore FilePath and ArtifactPath so that TranslateResourcePaths() can
	// rewrite local absolute paths to snapshot paths during destroy.
	b.Config.Workspace.FilePath = path.Join(snapshotPath, "src", "files")
	b.Config.Workspace.ArtifactPath = path.Join(snapshotPath, "src", "artifacts")
	return nil
}
