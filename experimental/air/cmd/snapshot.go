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

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

// Snapshot orchestrator: resolve → package+upload → sidecars, uploading via
// libs/filer. The Python CLI did this inline; here it's split into steps.

// snapshotResult holds the paths wired into the submit payload: the uploaded
// tarball and the optional provenance sidecars (empty when not produced).
type snapshotResult struct {
	CodeSourcePath string
	GitStatePath   string
	GitDiffPath    string
}

// repoSnapshotsSubdir is the per-user workspace location for cached tarballs, under
// the user's home. Volume uploads use remote_volume directly instead.
const repoSnapshotsSubdir = ".air/repo_snapshots"

// snapshotCodeSource packages and uploads the code_source snapshot, returning the
// paths to attach to the ai_runtime_task. userDir is the user's workspace home;
// funcDir is the run's launch directory (where sidecars land).
func snapshotCodeSource(ctx context.Context, w *databricks.WorkspaceClient, snap *snapshotSourceConfig, configPath, userDir, funcDir string) (snapshotResult, error) {
	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return snapshotResult{}, err
	}

	up, err := newSnapshotUploader(ctx, w, snap, userDir, funcDir, filepath.Base(repoPath))
	if err != nil {
		return snapshotResult{}, err
	}
	return runSnapshot(ctx, up, repoPath, snap)
}

// resolveRootPath resolves a snapshot root_path the way the Python normalize layer
// does: expand environment variables and ~, strip a leading "project_root/" (meaning
// "relative to the YAML file"), and resolve the rest against the config's directory.
// It then confirms the path exists and is a directory.
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

	// Resolve to an absolute path so the directory name (used for the tarball name
	// and archive prefix) is a real basename, not "." or a trailing relative segment.
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

// snapshotUploader splits the snapshot's two destinations: the tarball goes to a
// cache location (the user's repo_snapshots dir or a Volume), sidecars to the run's
// funcDir. tarBase/sidecarBase are the absolute roots, for reporting final paths.
type snapshotUploader struct {
	tarStore     filer.Filer
	sidecarStore filer.Filer
	tarBase      string
	sidecarBase  string
}

// runSnapshot resolves the packaging plan, uploads the tarball, then uploads the
// provenance sidecars. repoPath is the resolved root_path.
func runSnapshot(ctx context.Context, up snapshotUploader, repoPath string, snap *snapshotSourceConfig) (snapshotResult, error) {
	git := newGitRepo(repoPath)
	plan, err := resolveSnapshotPlan(ctx, git, snap.Git, snap.IncludePaths)
	if err != nil {
		return snapshotResult{}, err
	}

	dirName := filepath.Base(repoPath)

	tarName, err := uploadTarball(ctx, up, git, plan, repoPath, dirName)
	if err != nil {
		return snapshotResult{}, err
	}

	result := snapshotResult{CodeSourcePath: path.Join(up.tarBase, tarName)}

	// Provenance sidecars are best-effort: a git/upload hiccup here must not fail an
	// otherwise-valid submission. Non-git roots have no provenance to record.
	if plan.isGitRepo {
		result.GitStatePath, result.GitDiffPath = uploadSidecars(ctx, up, git, plan)
	}
	return result, nil
}

// uploadTarball packages the snapshot and uploads it, returning the tarball's name
// within the tar store. For git_archive it checks the cache first and skips
// packaging+upload on a hit. It writes the tarball to a temp file that is always
// cleaned up.
func uploadTarball(ctx context.Context, up snapshotUploader, git gitRepo, plan snapshotPlan, repoPath, dirName string) (string, error) {
	// git_archive is cacheable by (commit, include_paths); a hit means the identical
	// tarball is already uploaded, so packaging and upload are skipped entirely.
	if plan.mode == modeGitArchive {
		cacheKey := computeSnapshotCacheKey(plan.commitSHA, plan.includePaths)
		tarName := fmt.Sprintf("%s_%s.tar.gz", dirName, cacheKey[:16])
		if exists, err := fileExists(ctx, up.tarStore, tarName); err != nil {
			return "", err
		} else if exists {
			log.Debugf(ctx, "snapshot cache hit for %s at %s", shortSHA(plan.commitSHA), path.Join(up.tarBase, tarName))
			return tarName, nil
		}
		if err := packageAndUpload(ctx, up, tarName, func(out string) error {
			return createGitArchiveSnapshot(ctx, git, plan.commitSHA, out, dirName, plan.includePaths)
		}); err != nil {
			return "", err
		}
		return tarName, nil
	}

	// plain_tar is not cacheable (working-tree content isn't pinned to a SHA), so it
	// is timestamp-named to avoid clobbering a concurrent submission.
	tarName := fmt.Sprintf("%s_%s.tar.gz", dirName, time.Now().UTC().Format("20060102_150405"))
	if err := packageAndUpload(ctx, up, tarName, func(out string) error {
		return createPlainTarball(ctx, repoPath, out, plan.includePaths)
	}); err != nil {
		return "", err
	}
	return tarName, nil
}

// packageAndUpload writes the tarball via pkg into a temp file, then uploads it to
// tarName in the tar store. The temp file is always removed.
func packageAndUpload(ctx context.Context, up snapshotUploader, tarName string, pkg func(outputPath string) error) error {
	tmp, err := os.CreateTemp("", "air-snapshot-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp tarball: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	if err := pkg(tmpPath); err != nil {
		return err
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open tarball: %w", err)
	}
	defer f.Close()

	if err := up.tarStore.Write(ctx, tarName, f, filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		return fmt.Errorf("failed to upload snapshot to %s: %w", path.Join(up.tarBase, tarName), err)
	}
	return nil
}

// uploadSidecars builds and uploads the git_state.json and optional git_diff.patch
// provenance sidecars into the run's funcDir. It is best-effort: any failure logs a
// warning and returns whatever paths did upload (possibly none), never an error.
func uploadSidecars(ctx context.Context, up snapshotUploader, git gitRepo, plan snapshotPlan) (statePath, diffPath string) {
	mode := packagingModePlainTar
	pinnedTip := ""
	if plan.mode == modeGitArchive {
		mode = packagingModeGitArchive
		pinnedTip = plan.commitSHA
	}

	sidecar, err := buildGitStateSidecar(ctx, git, mode, pinnedTip, time.Now())
	if err != nil {
		log.Warnf(ctx, "skipping git provenance sidecar: %v", err)
		return "", ""
	}

	// Capture the dirty diff first so its status/path land in git_state.json.
	if sidecar.Dirty {
		status, diff := captureDirtyDiff(ctx, git, dirtyDiffSizeCapBytes, dirtyDiffTimeout)
		sidecar.DiffStatus = status
		if status == diffStatusCaptured {
			if err := up.sidecarStore.Write(ctx, gitDiffName, bytes.NewReader(diff), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
				log.Warnf(ctx, "failed to upload git diff sidecar: %v", err)
				sidecar.DiffStatus = diffStatusClean
			} else {
				diffPath = path.Join(up.sidecarBase, gitDiffName)
				sidecar.DiffPath = &diffPath
			}
		}
	}

	data, err := sidecar.marshal()
	if err != nil {
		log.Warnf(ctx, "failed to encode git state sidecar: %v", err)
		return "", diffPath
	}
	if err := up.sidecarStore.Write(ctx, gitStateName, bytes.NewReader(data), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		log.Warnf(ctx, "failed to upload git state sidecar: %v", err)
		return "", diffPath
	}
	return path.Join(up.sidecarBase, gitStateName), diffPath
}

// gitStateName and gitDiffName are the sidecar basenames read by the backend.
const (
	gitStateName = "git_state.json"
	gitDiffName  = "git_diff.patch"
)

// fileExists reports whether name exists in the store, treating fs.ErrNotExist as
// "no". Any other error propagates.
func fileExists(ctx context.Context, store filer.Filer, name string) (bool, error) {
	_, err := store.Stat(ctx, name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check snapshot cache: %w", err)
}

// newSnapshotUploader builds the uploader for a submission. The tarball store is a
// Volume (when remote_volume is set) or the user's repo_snapshots workspace dir;
// sidecars always go to the run's funcDir in the workspace.
func newSnapshotUploader(ctx context.Context, w *databricks.WorkspaceClient, snap *snapshotSourceConfig, userDir, funcDir, dirName string) (snapshotUploader, error) {
	sidecarStore, err := filer.NewWorkspaceFilesClient(w, funcDir)
	if err != nil {
		return snapshotUploader{}, err
	}

	if snap.RemoteVolume != nil {
		tarBase := strings.TrimRight(*snap.RemoteVolume, "/")
		tarStore, err := filer.NewFilesClient(ctx, w, tarBase)
		if err != nil {
			return snapshotUploader{}, err
		}
		return snapshotUploader{tarStore: tarStore, sidecarStore: sidecarStore, tarBase: tarBase, sidecarBase: funcDir}, nil
	}

	tarBase := path.Join(userDir, repoSnapshotsSubdir, dirName)
	tarStore, err := filer.NewWorkspaceFilesClient(w, tarBase)
	if err != nil {
		return snapshotUploader{}, err
	}
	return snapshotUploader{tarStore: tarStore, sidecarStore: sidecarStore, tarBase: tarBase, sidecarBase: funcDir}, nil
}
