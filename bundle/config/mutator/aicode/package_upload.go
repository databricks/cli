// Package aicode packages a local code directory referenced by an AI Runtime
// task's code_source_path and uploads it to the workspace during deploy.
//
// The SDK jobs.AiRuntimeTask.code_source_path field expects a workspace or UC
// volume path to an uploaded code archive; its doc comment states that the CLI
// is responsible for packaging the user's local code directory into that
// archive. This mutator implements that contract for DABs: when a user points
// code_source_path at a local directory, it snapshots the directory (git archive
// of HEAD when the tree is a clean git repo, otherwise a plain tar; .git and
// gitignored files excluded), uploads the archive to the user's workspace code
// snapshot cache, and rewrites the field to the resulting remote path so the
// deployed job runs against the uploaded code. Values that are already remote are
// left untouched.
//
// The snapshot approach mirrors PR #5897, which ported it from the Python air CLI
// (see snapshot_resolve.go / snapshot_package.go / snapshot_cachekey.go).
package aicode

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
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

// codeSourcePatterns are the config locations of an AI Runtime task's
// code_source_path, both as a direct task and nested under a for_each_task.
var codeSourcePatterns = []dyn.Pattern{
	dyn.NewPattern(
		dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(),
		dyn.Key("tasks"), dyn.AnyIndex(),
		dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"),
	),
	dyn.NewPattern(
		dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(),
		dyn.Key("tasks"), dyn.AnyIndex(),
		dyn.Key("for_each_task"), dyn.Key("task"),
		dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"),
	),
}

// codeSource is a single local code_source_path occurrence to package.
type codeSource struct {
	configPath dyn.Path
	location   dyn.Location
	// value is the raw code_source_path string as written in config.
	value string
}

func PackageAndUpload() bundle.Mutator {
	return &packageAndUpload{}
}

type packageAndUpload struct {
	// client is the filer used for uploads. When nil (the normal case) a filer
	// rooted at the code snapshot cache is built per code source. It is only set
	// in tests, to inject a recording filer.
	client filer.Filer
}

func (m *packageAndUpload) Name() string {
	return "aicode.PackageAndUpload"
}

func (m *packageAndUpload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	sources, diags := collectLocalCodeSources(b)
	if diags.HasError() {
		return diags
	}
	if len(sources) == 0 {
		return diags
	}

	userDir, err := userWorkspaceHome(b)
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	stagingDir, err := b.LocalStateDir(ctx, "ai_code_source")
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	// remotePaths maps each config location to the remote archive path it should
	// point to after upload. Built outside the Mutate closure so upload failures
	// are reported before any config is rewritten.
	remotePaths := make(map[string]string, len(sources))
	for _, cs := range sources {
		remote, err := m.packageOne(ctx, b, cs, stagingDir, userDir)
		if err != nil {
			diags = diags.Extend(diag.FromErr(err))
			return diags
		}
		remotePaths[cs.configPath.String()] = remote
	}

	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for _, cs := range sources {
			remote := remotePaths[cs.configPath.String()]
			root, err = dyn.SetByPath(root, cs.configPath, dyn.NewValue(remote, []dyn.Location{cs.location}))
			if err != nil {
				return root, fmt.Errorf("failed to update code_source_path %q to %q: %w", cs.value, remote, err)
			}
		}
		return root, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}

// repoSnapshotsSubdir is the per-user workspace location for cached code
// snapshots, under the user's home. It matches the Python air CLI (and PR #5897)
// so a snapshot is cached in a stable place that bundle deploy does not wipe.
//
// Deliberately NOT <artifact_path>/.internal: artifacts.CleanUp() deletes that
// directory at the start of every deploy, which would defeat the git-archive
// snapshot cache. Keeping snapshots under ~/.air/repo_snapshots lets an unchanged
// commit skip re-upload across deploys.
const repoSnapshotsSubdir = ".air/repo_snapshots"

// packageOne snapshots the local directory for a single code source, uploads the
// archive to the user's repo_snapshots dir (skipping the upload when an identical
// git-archive snapshot already exists), and returns the remote path the config
// should point to.
func (m *packageAndUpload) packageOne(ctx context.Context, b *bundle.Bundle, cs codeSource, stagingDir, userDir string) (string, error) {
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(cs.value))
	dirName := filepath.Base(localDir)

	uploadPath := path.Join(userDir, repoSnapshotsSubdir, dirName)
	client := m.client
	if client == nil {
		var err error
		client, err = filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), uploadPath)
		if err != nil {
			return "", err
		}
	}

	git := newGitRepo(localDir)
	plan, err := resolveSnapshotPlan(ctx, git)
	if err != nil {
		return "", fmt.Errorf("failed to resolve code_source_path %q: %w", cs.value, err)
	}

	// git_archive is cacheable by (commit, include_paths): a hit means the identical
	// tarball is already uploaded, so packaging and upload are skipped entirely.
	if plan.mode == modeGitArchive {
		cacheKey := computeSnapshotCacheKey(plan.commitSHA, plan.includePaths)
		archiveName := fmt.Sprintf("%s_%s.tar.gz", dirName, cacheKey[:16])
		remotePath := path.Join(uploadPath, archiveName)

		exists, err := archiveExists(ctx, client, archiveName)
		if err != nil {
			return "", err
		}
		if exists {
			log.Debugf(ctx, "code snapshot for %s already present at %s, skipping upload", shortSHA(plan.commitSHA), remotePath)
			return remotePath, nil
		}

		if err := packageAndUploadArchive(ctx, client, stagingDir, archiveName, func(out string) error {
			return createGitArchiveSnapshot(ctx, git, plan.commitSHA, out, dirName, plan.includePaths)
		}); err != nil {
			return "", err
		}
		return remotePath, nil
	}

	// plain_tar is not cacheable (working-tree content isn't pinned to a SHA), so it
	// is timestamp-named to avoid clobbering a concurrent deploy.
	archiveName := fmt.Sprintf("%s_%s.tar.gz", dirName, time.Now().UTC().Format("20060102_150405"))
	remotePath := path.Join(uploadPath, archiveName)
	if err := packageAndUploadArchive(ctx, client, stagingDir, archiveName, func(out string) error {
		return createPlainTarball(ctx, localDir, out, plan.includePaths)
	}); err != nil {
		return "", err
	}
	return remotePath, nil
}

// userWorkspaceHome returns the current user's workspace home directory
// (/Workspace/Users/<user>), the root under which code snapshots are cached.
func userWorkspaceHome(b *bundle.Bundle) (string, error) {
	u := b.Config.Workspace.CurrentUser
	if u == nil || u.User == nil || u.UserName == "" {
		return "", errors.New("unable to resolve code snapshot location: current user not set")
	}
	return "/Workspace/Users/" + u.UserName, nil
}

// archiveExists reports whether archiveName already exists in the store, treating
// fs.ErrNotExist as "no".
func archiveExists(ctx context.Context, client filer.Filer, archiveName string) (bool, error) {
	_, err := client.Stat(ctx, archiveName)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check for existing code snapshot %q: %w", archiveName, err)
}

// packageAndUploadArchive writes the tarball via pkg into a temp file under
// stagingDir, then uploads it as archiveName. The temp file is always removed.
func packageAndUploadArchive(ctx context.Context, client filer.Filer, stagingDir, archiveName string, pkg func(outputPath string) error) error {
	localArchive := filepath.Join(stagingDir, archiveName)
	defer os.Remove(localArchive)

	if err := pkg(localArchive); err != nil {
		return err
	}
	return libraries.UploadFile(ctx, localArchive, client)
}

// collectLocalCodeSources returns every AI Runtime task code_source_path that
// points at a local directory. Already-remote values are skipped.
func collectLocalCodeSources(b *bundle.Bundle) ([]codeSource, diag.Diagnostics) {
	var sources []codeSource
	var diags diag.Diagnostics

	for _, pattern := range codeSourcePatterns {
		err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
			return dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				value, ok := v.AsString()
				if !ok {
					return v, fmt.Errorf("expected string, got %s", v.Kind())
				}
				if !libraries.IsLocalPath(value) {
					return v, nil
				}
				sources = append(sources, codeSource{
					configPath: p,
					location:   v.Location(),
					value:      value,
				})
				return v, nil
			})
		})
		if err != nil {
			diags = diags.Extend(diag.FromErr(err))
		}
	}

	return sources, diags
}
