// Package aicode packages a local code directory referenced by an AI Runtime
// task's code_source_path and uploads it to the workspace during deploy.
//
// The SDK jobs.AiRuntimeTask.code_source_path field expects a workspace or UC
// volume path to an uploaded code archive; its doc comment states that the CLI
// is responsible for packaging the user's local code directory into that
// archive. This mutator implements that contract for DABs: when a user points
// code_source_path at a local directory, it packages the directory into a
// reproducible tarball (.git and gitignored files excluded), uploads the archive
// to the user's workspace code snapshot directory, and rewrites the field to the
// resulting remote path so the deployed job runs against the uploaded code. Values
// that are already remote are left untouched.
//
// The archive is content-addressed: its name embeds the SHA-256 of the
// (reproducible) tarball, so an unchanged code directory resolves to the same
// remote path across deploys and re-uploads are skipped (see snapshot_package.go).
package aicode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/log"
	libsync "github.com/databricks/cli/libs/sync"
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

	// remotePaths maps each config location to the remote archive path it should
	// point to after upload. Built outside the Mutate closure so upload failures
	// are reported before any config is rewritten.
	remotePaths := make(map[string]string, len(sources))
	for _, cs := range sources {
		remote, err := m.packageOne(ctx, b, cs, userDir)
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

// repoSnapshotsSubdir is the per-user workspace location for code snapshots,
// under the user's home. It matches the Python air CLI (and PR #5897) and is
// deliberately NOT <artifact_path>/.internal, which artifacts.CleanUp() deletes at
// the start of every deploy.
const repoSnapshotsSubdir = ".air/repo_snapshots"

// packageOne packages the local directory for a single code source into a
// reproducible tarball, uploads it to the user's repo_snapshots dir (skipping the
// upload when a content-identical archive already exists there), and returns the
// remote path the config should point to.
func (m *packageAndUpload) packageOne(ctx context.Context, b *bundle.Bundle, cs codeSource, userDir string) (string, error) {
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(cs.value))
	dirName := filepath.Base(localDir)

	// relBase is the code directory relative to the sync root, used both to scope the
	// sync file list to this directory and to re-base archive entry names under it.
	relBase, err := filepath.Rel(b.SyncRootPath, localDir)
	if err != nil {
		return "", fmt.Errorf("code_source_path %q: %w", cs.value, err)
	}
	relBase = filepath.ToSlash(relBase)

	files, err := codeSourceFiles(ctx, b, relBase)
	if err != nil {
		return "", fmt.Errorf("failed to list files for code_source_path %q: %w", cs.value, err)
	}

	uploadPath := path.Join(userDir, repoSnapshotsSubdir, dirName)
	client := m.client
	if client == nil {
		client, err = filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), uploadPath)
		if err != nil {
			return "", err
		}
	}

	// Build the archive in memory so its content hash can name the upload; the hash
	// is computed while gzipping, so this adds no extra pass over the files.
	var buf bytes.Buffer
	sha, err := buildCodeSnapshot(b.SyncRoot, relBase, files, dirName, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to package code_source_path %q: %w", cs.value, err)
	}

	archiveName := fmt.Sprintf("%s_%s.tar.gz", dirName, sha[:16])
	// The AI Runtime snapshot fetcher expects code_source_path in the legacy
	// "/Users/..." form (no "/Workspace" prefix), matching the Python air CLI. The
	// filer needs the "/Workspace/Users/..." form to upload, so upload to uploadPath
	// but record the de-prefixed path on the task.
	remotePath := strings.TrimPrefix(path.Join(uploadPath, archiveName), "/Workspace")

	// The archive is reproducible, so a matching name means identical content is
	// already uploaded: skip the upload and just point the config at it.
	if _, err := client.Stat(ctx, archiveName); err == nil {
		log.Debugf(ctx, "code snapshot already present at %s, skipping upload", remotePath)
		return remotePath, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("failed to check for existing code snapshot %q: %w", remotePath, err)
	}

	if err := client.Write(ctx, archiveName, &buf, filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		return "", fmt.Errorf("failed to upload code snapshot %q: %w", remotePath, err)
	}
	return remotePath, nil
}

// codeSourceFiles returns the files under the code directory (relBase, relative to
// the sync root) that should go into the snapshot. It reuses the bundle's sync
// options so the file list is filtered exactly like bundle file sync: .gitignore
// aware, plus the top-level sync.include/exclude globs. Scoping Paths to relBase
// restricts the walk (and the returned relative paths) to the code directory.
func codeSourceFiles(ctx context.Context, b *bundle.Bundle, relBase string) ([]fileset.File, error) {
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return nil, err
	}
	opts.Paths = []string{relBase}
	return libsync.GetFileList(ctx, *opts)
}

// userWorkspaceHome returns the current user's workspace home directory
// (/Workspace/Users/<user>), the root under which code snapshots are stored.
func userWorkspaceHome(b *bundle.Bundle) (string, error) {
	u := b.Config.Workspace.CurrentUser
	if u == nil || u.User == nil || u.UserName == "" {
		return "", errors.New("unable to resolve code snapshot location: current user not set")
	}
	return "/Workspace/Users/" + u.UserName, nil
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
