// Package aicode packages a local code directory referenced by an AI Runtime
// task's code_source_path and uploads it to the workspace during deploy.
//
// The SDK jobs.AiRuntimeTask.code_source_path field expects a workspace or UC
// volume path to an uploaded code archive; its doc comment states that the CLI
// is responsible for packaging the user's local code directory into that
// archive. This mutator implements that contract for DABs: when a user points
// code_source_path at a local directory, it tarballs the directory (gitignore
// aware, .git excluded), uploads the archive next to bundle libraries, and
// rewrites the field to the resulting remote path so the deployed job runs
// against the uploaded code. Values that are already remote are left untouched.
package aicode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/vfs"
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
	// client is the filer used for uploads. It defaults to the libraries filer
	// and is only overridden in tests.
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

	client, uploadPath, filerDiags := libraries.GetFilerForLibraries(ctx, b)
	diags = diags.Extend(filerDiags)
	if diags.HasError() {
		return diags
	}
	if m.client == nil {
		m.client = client
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
		remote, err := m.packageOne(ctx, b, cs, stagingDir, uploadPath)
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

// packageOne tarballs the local directory for a single code source, uploads it
// (skipping the upload if a content-identical archive already exists), and
// returns the remote path the config should point to.
func (m *packageAndUpload) packageOne(ctx context.Context, b *bundle.Bundle, cs codeSource, stagingDir, uploadPath string) (string, error) {
	localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(cs.value))
	dirName := filepath.Base(localDir)

	var buf bytes.Buffer
	sha, err := buildTarball(ctx, vfs.MustNew(localDir), dirName, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to package code_source_path %q: %w", cs.value, err)
	}

	// The SHA of the (reproducible) archive is embedded in the filename so an
	// unchanged code directory resolves to the same remote path across deploys
	// and the existence check below turns re-uploads into no-ops.
	archiveName := fmt.Sprintf("%s_%s.tar.gz", dirName, sha[:16])
	remotePath := path.Join(uploadPath, archiveName)

	if _, err := m.client.Stat(ctx, archiveName); err == nil {
		log.Debugf(ctx, "code snapshot %s already present, skipping upload", remotePath)
		return remotePath, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("failed to check for existing code snapshot %q: %w", remotePath, err)
	}

	// Stage the archive on disk so a large snapshot is not held in memory during
	// upload, matching how libraries are uploaded from a file.
	localArchive := filepath.Join(stagingDir, archiveName)
	if err := os.WriteFile(localArchive, buf.Bytes(), 0o600); err != nil {
		return "", err
	}

	if err := libraries.UploadFile(ctx, localArchive, m.client); err != nil {
		return "", err
	}

	return remotePath, nil
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
