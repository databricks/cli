package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// uploadProvenanceSidecars controls collection and upload of git_state.json
// and git_diff.patch during submission. Keep the implementation available, but
// turn it off for now because the additional local queries and WSFS writes add
// latency to AIR submissions.
const uploadProvenanceSidecars = false

// uploadSnapshot packages the code_source into a tarball and uploads it to the
// artifact store's .internal directory, returning the remote path to attach to the
// ai_runtime_task. Uploading reuses libs/filer for the workspace/volume write without
// depending on the bundle package (see uploadSnapshotTarball / snapshotUploadFiler).
func uploadSnapshot(ctx context.Context, w *databricks.WorkspaceClient, snap *snapshotSourceConfig, configPath, snapshotArtifactPath string, sidecarStore filer.Filer, sidecarBase string) (snapshotResult, error) {
	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return snapshotResult{}, err
	}

	// Resolve how to package before touching the tarball: git_archive (pinned commit) vs
	// plain_tar (working tree). Both are content-addressed, so an unchanged input reuses
	// the already-uploaded tarball instead of re-packaging and re-uploading.
	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repoPath), snap.Git, snap.IncludePaths)
	if err != nil {
		return snapshotResult{}, err
	}

	// remote_volume, when set, is a UC Volume path; snapshotUploadFiler routes
	// /Volumes destinations to a Files API filer.
	if snap.RemoteVolume != nil {
		snapshotArtifactPath = *snap.RemoteVolume
	}
	result, err := uploadSnapshotTarball(ctx, w, repoPath, plan, snapshotArtifactPath)
	if err != nil {
		return snapshotResult{}, err
	}

	// Upload git provenance sidecars (git_state.json / git_diff.patch) next to the
	// run's launch dir so the submitted commit + working-tree diff are inspectable.
	// Best-effort and git-only: any failure logs and leaves the paths empty rather
	// than failing an otherwise-valid submission.
	//
	// The sidecars are deliberately NOT bundled into the code tarball. The git_archive
	// tarball is content-addressed and cached by (commit, include_paths, root_path
	// subtree), so a second identical snapshot reuses it; but the sidecars vary per
	// run (git_state's timestamp, and git_diff captures the working tree at submit
	// time). Folding them in would force a distinct tarball per run (defeating the
	// cache) or serve a prior run's stale provenance on a cache hit. They also live in
	// the per-run launch dir,
	// not the shared artifact dir, so they don't accumulate. Keep them out of the tar.
	if uploadProvenanceSidecars && plan.isGitRepo {
		result.GitStatePath, result.GitDiffPath = uploadSnapshotSidecars(ctx, sidecarStore, sidecarBase, newGitRepo(repoPath), plan)
	}
	return result, nil
}

// uploadSnapshotSidecars writes the git_state.json provenance record — and, when the
// working tree is dirty, a captured git_diff.patch — into the run's launch dir via
// sidecarStore (rooted at sidecarBase, used only to report absolute paths). It is
// best-effort: every failure logs a warning and yields an empty path, never an error,
// so provenance capture cannot fail a submission.
func uploadSnapshotSidecars(ctx context.Context, sidecarStore filer.Filer, sidecarBase string, git gitRepo, plan snapshotPlan) (statePath, diffPath string) {
	mode := packagingModePlainTar
	pinnedTip := ""
	if plan.mode == modeGitArchive {
		mode = packagingModeGitArchive
		pinnedTip = plan.commitSHA
	}

	sidecar, err := buildGitStateSidecar(ctx, git, mode, pinnedTip, plan.hasUncommit, time.Now())
	if err != nil {
		log.Warnf(ctx, "skipping git provenance sidecar: %v", err)
		return "", ""
	}

	// Capture the dirty diff first so its status/path land in git_state.json.
	if sidecar.Dirty {
		status, diff := captureDirtyDiff(ctx, git, plan.includePaths, dirtyDiffSizeCapBytes, dirtyDiffTimeout)
		sidecar.DiffStatus = status
		if status == diffStatusCaptured {
			if err := sidecarStore.Write(ctx, gitDiffName, bytes.NewReader(diff), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
				log.Warnf(ctx, "failed to upload git diff sidecar: %v", err)
				sidecar.DiffStatus = diffStatusClean
			} else {
				diffPath = path.Join(sidecarBase, gitDiffName)
				sidecar.DiffPath = &diffPath
			}
		}
	}

	data, err := sidecar.marshal()
	if err != nil {
		log.Warnf(ctx, "failed to encode git state sidecar: %v", err)
		return "", diffPath
	}
	if err := sidecarStore.Write(ctx, gitStateName, bytes.NewReader(data), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		log.Warnf(ctx, "failed to upload git state sidecar: %v", err)
		return "", diffPath
	}
	return path.Join(sidecarBase, gitStateName), diffPath
}

// snapshotTarName resolves the content-addressed upload filename for the snapshot and,
// for plain_tar, the working-tree file listing used to build both the key and the tarball
// (nil for git_archive, which lists nothing locally). The name is <dirName>_<key>.tar.gz,
// keyed on (commit, include_paths, root_path subtree) for git_archive and on the
// working-tree metadata fingerprint for plain_tar, so an identical input
// reuses the same remote object (see the skip in uploadSnapshotTarball).
func snapshotTarName(ctx context.Context, repoPath string, plan snapshotPlan) (string, []snapshotFile, error) {
	dirName := filepath.Base(repoPath)
	if plan.mode == modeGitArchive {
		key := computeSnapshotCacheKey(plan.commitSHA, plan.includePaths, plan.subtreePrefix)
		return fmt.Sprintf("%s_%s.tar.gz", dirName, key[:16]), nil, nil
	}
	files, err := snapshotFiles(ctx, repoPath, plan.includePaths, plan.isGitRepo)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s_%s.tar.gz", dirName, computePlainTarKey(files)[:16]), files, nil
}

// packageSnapshot writes the snapshot to tarball per the resolved plan: `git archive`
// of the pinned commit for git_archive, else a plain tar of the pre-listed working-tree
// files (nil for git_archive).
func packageSnapshot(ctx context.Context, repoPath string, plan snapshotPlan, files []snapshotFile, tarball string) error {
	if plan.mode == modeGitArchive {
		return createGitArchiveSnapshot(ctx, newGitRepo(repoPath), plan.commitSHA, tarball, filepath.Base(repoPath), plan.includePaths, plan.subtreePrefix)
	}
	return createPlainTarball(ctx, repoPath, tarball, files)
}

// airArtifactInternalDir mirrors bundle/libraries.InternalDirName: CLI-uploaded
// artifacts live in a ".internal" subdirectory of the configured artifact path.
const airArtifactInternalDir = ".internal"

// uploadSnapshotTarball uploads the packaged snapshot to <artifactPath>/.internal and returns
// the remote code_source_path to attach to the ai_runtime_task. artifactPath is either
// the user's repo_snapshots directory or a configured UC Volume.
//
// The tarball name is content-addressed — by (commit, include_paths, root_path subtree)
// for git_archive and by the working-tree metadata fingerprint for plain_tar — so if
// the identical object already exists we skip packaging and upload entirely and reuse it.
func uploadSnapshotTarball(ctx context.Context, w *databricks.WorkspaceClient, repoPath string, plan snapshotPlan, artifactPath string) (snapshotResult, error) {
	f, uploadPath, err := snapshotUploadFiler(ctx, w, artifactPath)
	if err != nil {
		return snapshotResult{}, err
	}

	tarName, files, err := snapshotTarName(ctx, repoPath, plan)
	if err != nil {
		return snapshotResult{}, err
	}
	// code_source_path is content-addressed by tarName, so it is the same whether we
	// upload the bytes now or reuse an object already in the store.
	remote := path.Join(uploadPath, tarName)

	exists, err := snapshotExists(ctx, f, tarName)
	if err != nil {
		return snapshotResult{}, err
	}
	if exists {
		log.Debugf(ctx, "snapshot upload skipped; reusing %s", remote)
		return snapshotResult{CodeSourcePath: remote}, nil
	}

	tmp, err := os.MkdirTemp("", "air-snapshot-*")
	if err != nil {
		return snapshotResult{}, err
	}
	defer os.RemoveAll(tmp)

	tarball := filepath.Join(tmp, tarName)
	if err := packageSnapshot(ctx, repoPath, plan, files, tarball); err != nil {
		return snapshotResult{}, err
	}

	file, err := os.Open(tarball)
	if err != nil {
		return snapshotResult{}, err
	}
	defer file.Close()
	cmdio.LogProgress(ctx, fmt.Sprintf("Uploading %s...", tarName))
	if err := f.Write(ctx, tarName, file, filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		return snapshotResult{}, fmt.Errorf("failed to upload snapshot %s: %w", tarName, err)
	}
	return snapshotResult{CodeSourcePath: remote}, nil
}

// snapshotUploadFiler returns a filer rooted at <artifactPath>/.internal plus that
// upload path. It routes /Volumes paths to a Files API filer and everything else to a
// workspace files filer, matching bundle artifact-upload resolution without depending
// on the bundle package.
func snapshotUploadFiler(ctx context.Context, w *databricks.WorkspaceClient, artifactPath string) (filer.Filer, string, error) {
	if artifactPath == "" {
		return nil, "", errors.New("remote artifact path not configured")
	}
	uploadPath := path.Join(artifactPath, airArtifactInternalDir)
	// A bare workspace path (e.g. /Users/...) is served under /Workspace by the backend.
	if !strings.HasPrefix(uploadPath, "/Workspace") && !strings.HasPrefix(uploadPath, "/Volumes") {
		uploadPath = "/Workspace" + uploadPath
	}
	if strings.HasPrefix(artifactPath, "/Volumes/") {
		f, err := filer.NewFilesClient(ctx, w, uploadPath)
		return f, uploadPath, err
	}
	f, err := filer.NewWorkspaceFilesClient(w, uploadPath)
	return f, uploadPath, err
}

// snapshotExists reports whether name already exists in the artifact store, used to
// short-circuit a content-addressed upload (either mode). A not-found is a clean miss
// (false, nil); any other error is surfaced.
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
