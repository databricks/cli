package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
)

type loadState struct{}

// LoadState reads workspace.snapshot_path from the local deployment.json and
// sets the snapshot-derived workspace paths. Missing or empty state is treated
// as a no-op so destroy can proceed against bundles deployed before this
// feature was added.
func LoadState() bundle.Mutator {
	return &loadState{}
}

func (s *loadState) Name() string { return "snapshot.LoadState" }

func (s *loadState) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	localPath := filepath.Join(b.GetLocalStateDir(ctx), deploy.DeploymentStateFileName)
	data, err := os.ReadFile(localPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return diag.FromErr(err)
	}

	if err == nil {
		var state struct {
			SnapshotPath string `json:"snapshot_path"`
		}
		if jsonErr := json.Unmarshal(data, &state); jsonErr != nil {
			return diag.FromErr(jsonErr)
		}
		if state.SnapshotPath != "" {
			applySnapshotPath(b, state.SnapshotPath)
			return nil
		}
	}

	// Local deployment.json is missing or was from a non-immutable deploy — fall
	// back to the remote copy so destroy works on a fresh clone or a different machine.
	return s.loadFromRemote(ctx, b)
}

func (s *loadState) loadFromRemote(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return diag.FromErr(err)
	}

	r, err := f.Read(ctx, deploy.DeploymentStateFileName)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	defer r.Close()

	var state struct {
		SnapshotPath string `json:"snapshot_path"`
	}
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return diag.FromErr(err)
	}

	if state.SnapshotPath != "" {
		applySnapshotPath(b, state.SnapshotPath)
	}
	return nil
}

func applySnapshotPath(b *bundle.Bundle, snapshotPath string) {
	b.Config.Workspace.SnapshotPath = snapshotPath
	// Restore FilePath and ArtifactPath for other callers (permissions checks, etc.).
	// The resource paths themselves are resolved later via ResolveVariableReferencesOnlyResources("workspace").
	b.Config.Workspace.FilePath = path.Join(snapshotPath, "src", "files")
	b.Config.Workspace.ArtifactPath = path.Join(snapshotPath, "src", "artifacts")
}
