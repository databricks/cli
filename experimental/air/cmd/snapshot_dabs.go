package aircmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// snapshotViaDABsUpload packages the code_source into a tarball and uploads it using
// DABs' artifact-upload plumbing (the same path a bundle uses for a file-valued
// code_source_path), returning the remote path to attach to the ai_runtime_task.
//
// The packaging + upload logic is CLI-owned (this file, OWNERS = us); it only reuses
// DABs' libraries.ReplaceWithRemotePath + libraries.Upload as the uploader so we do
// not reimplement workspace/volume upload. A minimal in-memory bundle carries the
// local tarball path as code_source_path; ReplaceWithRemotePath rewrites it to the
// artifact .internal path and Upload pushes the bytes.
func snapshotViaDABsUpload(ctx context.Context, w *databricks.WorkspaceClient, snap *snapshotSourceConfig, configPath string) (snapshotResult, error) {
	if snap.RemoteVolume != nil {
		return snapshotResult{}, errors.New("code_source.snapshot.remote_volume is not yet supported")
	}
	if snap.Git != nil {
		return snapshotResult{}, errors.New("code_source.snapshot.git is not yet supported on this path")
	}

	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return snapshotResult{}, err
	}

	tarball, cleanup, err := buildSnapshotTarball(ctx, repoPath, snap.IncludePaths)
	if err != nil {
		return snapshotResult{}, err
	}
	defer cleanup()

	return uploadSnapshotViaDABs(ctx, w, tarball)
}

// buildSnapshotTarball writes a plain-tar snapshot of repoPath to a temp file named
// <dirName>.tar.gz (the basename becomes the uploaded filename), returning the path
// and a cleanup func.
func buildSnapshotTarball(ctx context.Context, repoPath string, includePaths []string) (string, func(), error) {
	noop := func() {}
	tmp, err := os.MkdirTemp("", "air-snapshot-*")
	if err != nil {
		return "", noop, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	tarball := filepath.Join(tmp, filepath.Base(repoPath)+".tar.gz")
	if err := createPlainTarball(ctx, repoPath, tarball, includePaths); err != nil {
		cleanup()
		return "", noop, err
	}
	return tarball, cleanup, nil
}

// uploadSnapshotViaDABs uploads a local tarball through DABs' artifact-upload
// machinery and returns its remote code_source_path. It builds a minimal bundle whose
// only artifact is the tarball (as a file-valued code_source_path), rewrites the field
// to the remote .internal path, and uploads the bytes.
func uploadSnapshotViaDABs(ctx context.Context, w *databricks.WorkspaceClient, tarball string) (snapshotResult, error) {
	base, err := userWorkspaceDir(ctx, w)
	if err != nil {
		return snapshotResult{}, err
	}
	// Upload under the user's repo_snapshots dir (not the default bundle artifact_path,
	// which a deploy would clean up); ReplaceWithRemotePath appends /.internal.
	artifactPath := path.Join(base, ".air", "repo_snapshots")

	b := &bundle.Bundle{
		BundleRootPath: filepath.Dir(tarball),
		BundleRoot:     vfs.MustNew(filepath.Dir(tarball)),
		SyncRootPath:   filepath.Dir(tarball),
		SyncRoot:       vfs.MustNew(filepath.Dir(tarball)),
		Config: config.Root{
			Bundle:    config.Bundle{Target: "default"},
			Workspace: config.Workspace{ArtifactPath: artifactPath},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"air": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{{
								TaskKey: "air",
								// Relative to SyncRootPath (the tarball's dir); collectLocalLibraries
								// joins it back and uploads the file.
								AiRuntimeTask: &jobs.AiRuntimeTask{CodeSourcePath: filepath.Base(tarball)},
							}},
						},
					},
				},
			},
		},
	}
	b.SetWorkpaceClient(w)
	if err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) { return v, nil }); err != nil {
		return snapshotResult{}, err
	}

	// Rewrite code_source_path to its remote .internal path (returns the upload set),
	// then upload the bytes.
	libs, diags := libraries.ReplaceWithRemotePath(ctx, b)
	if diags.HasError() {
		return snapshotResult{}, diags.Error()
	}
	if diags := bundle.Apply(ctx, b, libraries.Upload(libs)); diags.HasError() {
		return snapshotResult{}, diags.Error()
	}

	remote, err := readCodeSourcePath(b)
	if err != nil {
		return snapshotResult{}, err
	}
	return snapshotResult{CodeSourcePath: remote}, nil
}

// readCodeSourcePath returns the (rewritten) code_source_path from the bundle config.
func readCodeSourcePath(b *bundle.Bundle) (string, error) {
	v, err := dyn.GetByPath(b.Config.Value(),
		dyn.MustPathFromString("resources.jobs.air.tasks[0].ai_runtime_task.code_source_path"))
	if err != nil {
		return "", fmt.Errorf("code snapshot was not packaged: %w", err)
	}
	s, ok := v.AsString()
	if !ok {
		return "", fmt.Errorf("unexpected code_source_path value %v", v.AsAny())
	}
	return s, nil
}
