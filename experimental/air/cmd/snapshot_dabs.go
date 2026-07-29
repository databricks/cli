package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
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
	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return snapshotResult{}, err
	}

	// Resolve how to package before touching the tarball: git_archive (pinned commit,
	// cacheable) vs plain_tar (working tree, not cacheable).
	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repoPath), snap.Git, snap.IncludePaths)
	if err != nil {
		return snapshotResult{}, err
	}

	// remote_volume, when set, is a UC Volume path; DABs' artifact uploader handles
	// /Volumes destinations natively (GetFilerForLibraries → filerForVolume).
	remoteVolume := ""
	if snap.RemoteVolume != nil {
		remoteVolume = *snap.RemoteVolume
	}
	return uploadSnapshotViaDABs(ctx, w, repoPath, plan, remoteVolume)
}

// snapshotTarballName is the uploaded filename for the snapshot. It is deterministic
// for git_archive — <dirName>_<cacheKey>.tar.gz keyed on (commit, include_paths) — so
// an identical commit reuses the same remote object (see the cache check below). For
// plain_tar it is timestamped so concurrent submissions of the same directory don't
// clobber each other's upload (working-tree content isn't pinned to a SHA, so it
// can't be content-addressed).
func snapshotTarballName(plan snapshotPlan, dirName string) string {
	if plan.mode == modeGitArchive {
		key := computeSnapshotCacheKey(plan.commitSHA, plan.includePaths)
		return fmt.Sprintf("%s_%s.tar.gz", dirName, key[:16])
	}
	return fmt.Sprintf("%s_%s.tar.gz", dirName, time.Now().UTC().Format("20060102_150405"))
}

// packageSnapshot writes the snapshot to tarball per the resolved plan: `git archive`
// of the pinned commit for git_archive, else a plain tar of the working tree.
func packageSnapshot(ctx context.Context, repoPath string, plan snapshotPlan, tarball string) error {
	dirName := filepath.Base(repoPath)
	if plan.mode == modeGitArchive {
		return createGitArchiveSnapshot(ctx, newGitRepo(repoPath), plan.commitSHA, tarball, dirName, plan.includePaths)
	}
	return createPlainTarball(ctx, repoPath, tarball, plan.includePaths)
}

// uploadSnapshotViaDABs uploads the snapshot through DABs' artifact-upload machinery
// and returns its remote code_source_path. It builds a minimal bundle whose only
// artifact is the tarball (as a file-valued code_source_path), rewrites the field to
// the remote .internal path, and uploads the bytes. When remoteVolume is set the
// tarball goes to that UC Volume; otherwise to the user's repo_snapshots dir.
//
// git_archive snapshots are cacheable: the tarball name is content-addressed by
// (commit, include_paths), so if the identical object is already uploaded we skip
// packaging and upload entirely and just reuse the remote path.
func uploadSnapshotViaDABs(ctx context.Context, w *databricks.WorkspaceClient, repoPath string, plan snapshotPlan, remoteVolume string) (snapshotResult, error) {
	// artifactPath is where DABs uploads the tarball; GetFilerForLibraries routes to
	// a Workspace or Volume filer based on its prefix, then appends /.internal.
	artifactPath := remoteVolume
	if artifactPath == "" {
		base, err := userWorkspaceDir(ctx, w)
		if err != nil {
			return snapshotResult{}, err
		}
		// The user's repo_snapshots dir (not the default bundle artifact_path, which a
		// deploy would clean up).
		artifactPath = path.Join(base, ".air", "repo_snapshots")
	}

	tmp, err := os.MkdirTemp("", "air-snapshot-*")
	if err != nil {
		return snapshotResult{}, err
	}
	defer os.RemoveAll(tmp)

	tarName := snapshotTarballName(plan, filepath.Base(repoPath))

	b := &bundle.Bundle{
		BundleRootPath: tmp,
		BundleRoot:     vfs.MustNew(tmp),
		SyncRootPath:   tmp,
		SyncRoot:       vfs.MustNew(tmp),
		Config: config.Root{
			Bundle:    config.Bundle{Target: "default"},
			Workspace: config.Workspace{ArtifactPath: artifactPath},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"air": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{{
								TaskKey: "air",
								// Relative to SyncRootPath (the temp dir); collectLocalLibraries
								// joins it back and uploads the file.
								AiRuntimeTask: &jobs.AiRuntimeTask{CodeSourcePath: tarName},
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

	// git_archive is cacheable by (commit, include_paths): if the identical tarball is
	// already uploaded, skip packaging + upload and reuse it. Only the config-path
	// rewrite (ReplaceWithRemotePath) runs — no bytes move.
	if plan.mode == modeGitArchive {
		f, uploadPath, diags := libraries.GetFilerForLibraries(ctx, b)
		if diags.HasError() {
			return snapshotResult{}, diags.Error()
		}
		exists, err := snapshotExists(ctx, f, tarName)
		if err != nil {
			return snapshotResult{}, err
		}
		if exists {
			if _, diags := libraries.ReplaceWithRemotePath(ctx, b); diags.HasError() {
				return snapshotResult{}, diags.Error()
			}
			remote, err := readCodeSourcePath(b)
			if err != nil {
				return snapshotResult{}, err
			}
			log.Debugf(ctx, "snapshot cache hit for %s at %s", shortSHA(plan.commitSHA), path.Join(uploadPath, tarName))
			return snapshotResult{CodeSourcePath: remote}, nil
		}
	}

	// Cache miss (or plain_tar): package the tarball locally, then upload the bytes.
	if err := packageSnapshot(ctx, repoPath, plan, filepath.Join(tmp, tarName)); err != nil {
		return snapshotResult{}, err
	}

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

// snapshotExists reports whether name already exists in the artifact store, used to
// short-circuit a cacheable git_archive upload. A not-found is a clean miss (false,
// nil); any other error is surfaced.
func snapshotExists(ctx context.Context, store filer.Filer, name string) (bool, error) {
	_, err := store.Stat(ctx, name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check snapshot cache: %w", err)
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
