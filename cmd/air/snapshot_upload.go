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
	"github.com/databricks/cli/libs/tarpack"
	"github.com/databricks/databricks-sdk-go"
)

// uploadProvenanceSidecars controls collection and upload of git_state.json
// and git_diff.patch during submission. Keep the implementation available, but
// turn it off for now because the additional local queries and WSFS writes add
// latency to AIR submissions.
const uploadProvenanceSidecars = false

// snapshotLargeSourceBytes is the uncompressed size of the shipped files at or
// above which the non-git snapshot notice suggests narrowing the selection. The
// value is an arbitrary default, not derived from measurements; tune it freely.
const snapshotLargeSourceBytes = 64 << 20

// snapshotSkippedPathsShown caps how many skipped paths the warning names.
const snapshotSkippedPathsShown = 5

// humanBytes formats a byte count using binary (1024-based) units. It is a copy
// of humanBytes in cmd/fs/upload_progress.go, which is unexported there.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < len(units)-1 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(n)/float64(div), units[exp])
}

// shippedSnapshotFiles applies the plain tarball's entry rule (tarpack.Archives)
// to a listing: it returns how many entries the tarball will hold, their
// uncompressed content bytes (symlinks carry none), and the listed paths it will
// skip. ls-files lists a nested git repository as one directory entry, so that
// repository's files are skipped with it.
func shippedSnapshotFiles(files []snapshotFile) (count int, sourceBytes int64, skipped []string) {
	for _, f := range files {
		mode := fs.FileMode(f.mode)
		if !tarpack.Archives(mode) {
			skipped = append(skipped, filepath.ToSlash(f.rel))
			continue
		}
		count++
		if mode.IsRegular() {
			sourceBytes += f.size
		}
	}
	return count, sourceBytes, skipped
}

// plainTarNoticeLines renders the non-git snapshot disclosure. The size is the
// uncompressed shipped bytes, which include_paths and .gitignore control; the
// compressed tarball size stays telemetry-only. reuse marks a cache hit, where
// nothing is packaged or uploaded, so the size tip is omitted.
func plainTarNoticeLines(repoPath string, count int, sourceBytes int64, reuse bool) []string {
	base := filepath.ToSlash(repoPath)
	total := humanBytes(sourceBytes)
	if reuse {
		return []string{fmt.Sprintf("No git repository at %s; reusing the cached snapshot (%d file(s), %s uncompressed).", base, count, total)}
	}
	lines := []string{fmt.Sprintf("No git repository at %s; packaging %d file(s) (%s uncompressed) from the working tree.", base, count, total)}
	if sourceBytes >= snapshotLargeSourceBytes {
		lines = append(lines, "Tip: limit the snapshot with code_source.include_paths or a .gitignore in the directory.")
	}
	return lines
}

// skippedSnapshotWarning names up to snapshotSkippedPathsShown listed paths that
// the tarball leaves out.
func skippedSnapshotWarning(skipped []string) string {
	shown := skipped
	more := ""
	if len(shown) > snapshotSkippedPathsShown {
		shown = shown[:snapshotSkippedPathsShown]
		more = fmt.Sprintf(" (and %d more)", len(skipped)-snapshotSkippedPathsShown)
	}
	return fmt.Sprintf("Warning: the snapshot leaves out %d listed path(s) that are not regular files or symlinks, such as nested git repositories: %s%s.", len(skipped), strings.Join(shown, ", "), more)
}

// preparedSnapshot is a code_source snapshot resolved up to, but not including,
// packaging: the plan, the content-addressed tarball name, the plain_tar listing
// and whether the store already holds the tarball. submitWorkload resolves it
// before the staging spinner starts so the snapshot notice prints first.
type preparedSnapshot struct {
	repoPath string
	plan     snapshotPlan
	store    filer.Filer
	tarName  string
	remote   string
	// files is the plain_tar listing; nil for git_archive.
	files  []snapshotFile
	exists bool
	// result carries the telemetry fields known before packaging, and is
	// returned on a resolution failure after the plan is known.
	result snapshotResult
}

// uploadSnapshot packages the code_source into a tarball and uploads it to the
// artifact store's .internal directory, returning the remote path to attach to the
// ai_runtime_task. Uploading reuses libs/filer for the workspace/volume write without
// depending on the bundle package (see uploadSnapshotTarball / snapshotUploadFiler).
func uploadSnapshot(ctx context.Context, w *databricks.WorkspaceClient, snap *snapshotSourceConfig, configPath, snapshotArtifactPath string, sidecarStore filer.Filer, sidecarBase string) (snapshotResult, error) {
	p, err := prepareSnapshot(ctx, w, snap, configPath, snapshotArtifactPath)
	if err != nil {
		return p.result, err
	}
	return p.upload(ctx, sidecarStore, sidecarBase)
}

// prepareSnapshot resolves the root path, packaging plan and tarball name, and
// checks whether the tarball is already uploaded.
func prepareSnapshot(ctx context.Context, w *databricks.WorkspaceClient, snap *snapshotSourceConfig, configPath, snapshotArtifactPath string) (preparedSnapshot, error) {
	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return preparedSnapshot{}, err
	}

	// Resolve how to package before touching the tarball: git_archive (pinned commit) vs
	// plain_tar (working tree). Both are content-addressed, so an unchanged input reuses
	// the already-uploaded tarball instead of re-packaging and re-uploading.
	plan, err := resolveSnapshotPlan(ctx, newGitRepo(repoPath), snap.Git, snap.IncludePaths)
	if err != nil {
		return preparedSnapshot{}, err
	}

	// remote_volume, when set, is a UC Volume path; snapshotUploadFiler routes
	// /Volumes destinations to a Files API filer.
	if snap.RemoteVolume != nil {
		snapshotArtifactPath = *snap.RemoteVolume
	}
	return prepareSnapshotTarball(ctx, w, repoPath, plan, snapshotArtifactPath)
}

// announceSnapshot tells the user what a non-git snapshot ships. The skipped-path
// warning prints in every output mode, since those contents are silently missing
// from the run; the notice is progress output and prints only when showNotice is
// set (text mode). Git snapshots are scoped by git and get neither.
func announceSnapshot(ctx context.Context, p preparedSnapshot, showNotice bool) {
	if p.plan.mode != modePlainTar || p.plan.isGitRepo {
		return
	}
	count, sourceBytes, skipped := shippedSnapshotFiles(p.files)
	if len(skipped) > 0 {
		cmdio.LogString(ctx, skippedSnapshotWarning(skipped))
	}
	if !showNotice {
		return
	}
	for _, line := range plainTarNoticeLines(p.repoPath, count, sourceBytes, p.exists) {
		cmdio.LogProgress(ctx, line)
	}
}

// upload packages and uploads the prepared snapshot (unless it is already in the
// store) and returns its remote code_source_path.
func (p preparedSnapshot) upload(ctx context.Context, sidecarStore filer.Filer, sidecarBase string) (snapshotResult, error) {
	result, err := p.uploadTarball(ctx)
	if err != nil {
		return result, err
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
	if uploadProvenanceSidecars && p.plan.isGitRepo {
		result.GitStatePath, result.GitDiffPath = uploadSnapshotSidecars(ctx, sidecarStore, sidecarBase, newGitRepo(p.repoPath), p.plan)
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

// countSnapshotFiles runs the snapshot file selection without packaging or
// uploading, so --dry-run fails on the same selection errors as a submission.
func countSnapshotFiles(ctx context.Context, snap *snapshotSourceConfig, configPath string) (int, error) {
	repoPath, err := resolveRootPath(ctx, snap.RootPath, filepath.Dir(configPath))
	if err != nil {
		return 0, err
	}
	git := newGitRepo(repoPath)
	plan, err := resolveSnapshotPlan(ctx, git, snap.Git, snap.IncludePaths)
	if err != nil {
		return 0, err
	}
	if plan.mode == modeGitArchive {
		return git.countFilesAtCommit(ctx, plan.commitSHA, plan.subtreePrefix, plan.includePaths)
	}
	files, err := snapshotFiles(ctx, repoPath, plan.includePaths, plan.isGitRepo)
	if err != nil {
		return 0, err
	}
	count, _, _ := shippedSnapshotFiles(files)
	return count, nil
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
func uploadSnapshotTarball(ctx context.Context, w *databricks.WorkspaceClient, repoPath string, plan snapshotPlan, artifactPath string) (snapshotResult, error) {
	p, err := prepareSnapshotTarball(ctx, w, repoPath, plan, artifactPath)
	if err != nil {
		return p.result, err
	}
	return p.uploadTarball(ctx)
}

// prepareSnapshotTarball resolves the upload store and the content-addressed tarball
// name — by (commit, include_paths, root_path subtree) for git_archive and by the
// working-tree metadata fingerprint for plain_tar — and checks whether the identical
// object already exists, in which case uploadTarball skips packaging and upload.
func prepareSnapshotTarball(ctx context.Context, w *databricks.WorkspaceClient, repoPath string, plan snapshotPlan, artifactPath string) (preparedSnapshot, error) {
	p := preparedSnapshot{
		repoPath: repoPath,
		plan:     plan,
		result:   snapshotResult{UsesGit: &plan.isGitRepo, PackagingMode: &plan.mode},
	}
	f, uploadPath, err := snapshotUploadFiler(ctx, w, artifactPath)
	if err != nil {
		return p, err
	}
	p.store = f

	p.tarName, p.files, err = snapshotTarName(ctx, repoPath, plan)
	if err != nil {
		return p, err
	}
	// code_source_path is content-addressed by tarName, so it is the same whether we
	// upload the bytes now or reuse an object already in the store.
	p.remote = path.Join(uploadPath, p.tarName)

	p.exists, err = snapshotExists(ctx, f, p.tarName)
	return p, err
}

// uploadTarball packages and uploads the prepared tarball, or reuses the stored one.
func (p preparedSnapshot) uploadTarball(ctx context.Context) (snapshotResult, error) {
	result := p.result
	tarName := p.tarName
	if p.exists {
		log.Debugf(ctx, "snapshot upload skipped; reusing %s", p.remote)
		result.CodeSourcePath = p.remote
		result.PackagingDurationMs = new(int64(0))
		result.UploadDurationMs = new(int64(0))
		return result, nil
	}

	tmp, err := os.MkdirTemp("", "air-snapshot-*")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(tmp)

	tarball := filepath.Join(tmp, tarName)
	packagingStart := time.Now()
	err = packageSnapshot(ctx, p.repoPath, p.plan, p.files, tarball)
	result.PackagingDurationMs = new(time.Since(packagingStart).Milliseconds())
	if err != nil {
		return result, err
	}

	file, err := os.Open(tarball)
	if err != nil {
		return result, err
	}
	defer file.Close()
	// Size collection is best-effort and must not prevent a valid upload.
	if info, err := file.Stat(); err == nil {
		size := info.Size()
		result.SizeBytes = &size
	} else {
		log.Debugf(ctx, "failed to measure code snapshot: %v", err)
	}
	cmdio.LogProgress(ctx, fmt.Sprintf("Uploading %s...", tarName))
	uploadStart := time.Now()
	err = p.store.Write(ctx, tarName, file, filer.OverwriteIfExists, filer.CreateParentDirectories)
	result.UploadDurationMs = new(time.Since(uploadStart).Milliseconds())
	if err != nil {
		return result, fmt.Errorf("failed to upload snapshot %s: %w", tarName, err)
	}
	result.CodeSourcePath = p.remote
	return result, nil
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
