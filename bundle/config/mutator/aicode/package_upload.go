// Package aicode packages a local code_source_path directory into a
// content-addressed tarball at deploy, uploads it to the user's workspace, and
// rewrites code_source_path to the remote archive. Already-remote values are left
// untouched. The archive name embeds its SHA-256, so unchanged code skips re-upload.
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

// codeSourcePatterns locate code_source_path on a direct task and under a for_each_task.
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
	value      string // raw code_source_path string as written in config
}

func PackageAndUpload() bundle.Mutator {
	return &packageAndUpload{}
}

type packageAndUpload struct {
	client filer.Filer // nil in normal use (a filer is built per code source); set only in tests
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

	// Upload all sources before rewriting any config, so an upload failure leaves
	// the config untouched.
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

// repoSnapshotsSubdir holds code snapshots under the user's home. Deliberately not
// <artifact_path>/.internal, which artifacts.CleanUp() wipes each deploy (matches the Python air CLI).
const repoSnapshotsSubdir = ".air/repo_snapshots"

// packageOne tarballs one code source, uploads it to repo_snapshots (skipping the
// upload when a content-identical archive is already there), and returns its remote path.
func (m *packageAndUpload) packageOne(ctx context.Context, b *bundle.Bundle, cs codeSource, userDir string) (string, error) {
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(cs.value))
	dirName := filepath.Base(localDir)

	// relBase scopes the sync file list to the code dir and re-bases archive entries under it.
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

	// Build in memory so the content hash (computed while gzipping) can name the upload.
	var buf bytes.Buffer
	sha, err := buildCodeSnapshot(b.SyncRoot, relBase, files, dirName, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to package code_source_path %q: %w", cs.value, err)
	}

	archiveName := fmt.Sprintf("%s_%s.tar.gz", dirName, sha[:16])
	// The AI Runtime fetcher wants the legacy "/Users/..." form (no "/Workspace"), while
	// the filer uploads via uploadPath; record the de-prefixed path on the task.
	remotePath := strings.TrimPrefix(path.Join(uploadPath, archiveName), "/Workspace")

	// A matching name means identical content is already uploaded (the archive is reproducible).
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

// codeSourceFiles lists the files under relBase (relative to the sync root) for the
// snapshot, filtered like bundle file sync: .gitignore-aware plus sync.include/exclude.
func codeSourceFiles(ctx context.Context, b *bundle.Bundle, relBase string) ([]fileset.File, error) {
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return nil, err
	}
	fl, err := libsync.NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, []string{relBase}, opts.Include, opts.Exclude)
	if err != nil {
		return nil, err
	}
	return fl.Files(ctx)
}

// userWorkspaceHome returns /Workspace/Users/<user>, the root for code snapshots.
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
