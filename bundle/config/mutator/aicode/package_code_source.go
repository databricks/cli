// Package aicode routes an AI Runtime task's local-directory code_source_path through
// the standard artifact path: it synthesizes a `tgz` artifact that packages the
// directory and rewrites code_source_path to the tarball the artifact builds. Remote
// values and local files (a pre-built tarball from an explicit `artifacts` block) are
// left untouched.
//
// It runs before artifacts.Prepare, so the synthesized artifact is prepared, built, and
// uploaded by the normal artifact pipeline — there is no sync-root overlay. Because it
// only edits config (no packaging or workspace writes), it is safe in the initialize
// phase; the tarball itself is produced later by artifacts.Build.
package aicode

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

// codeSourcePattern is the config location of an AI Runtime task's code_source_path. It
// matches a direct task only — the same scope aicode.Validate operates on.
var codeSourcePattern = dyn.NewPattern(
	dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(),
	dyn.Key("tasks"), dyn.AnyIndex(),
	dyn.Key("ai_runtime_task"), dyn.Key("code_source_path"),
)

// codeArtifactOutputDir is where synthesized tgz artifacts write their tarball. It lives
// under .databricks (transient, not synced) so the built file is uploaded once via the
// artifact path and never swept into a sync or into the archive it produces.
const codeArtifactOutputDir = ".databricks/air_code_source"

// codeSource is a single local code_source_path occurrence to package.
type codeSource struct {
	configPath dyn.Path
	location   dyn.Location
	// value is the raw code_source_path string as written in config.
	value string
}

func PackageCodeSource() bundle.Mutator {
	return &packageCodeSource{}
}

type packageCodeSource struct{}

func (m *packageCodeSource) Name() string {
	return "aicode.PackageCodeSource"
}

func (m *packageCodeSource) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	sources, diags := collectLocalCodeSources(b)
	if diags.HasError() || len(sources) == 0 {
		return diags
	}

	// artifacts maps the synthesized artifact name to its spec; outputs maps each
	// code_source_path config location to the tarball it should point at. The
	// code_source_path and the artifact's `files` output share the same local path —
	// that shared path is what links them when libraries upload rewrites both to the
	// same remote location.
	artifacts := make(map[string]*config.Artifact, len(sources))
	outputs := make(map[string]string, len(sources))
	// keyDir records which relDir each artifact key was derived from. artifactKey
	// sanitizes non-alphanumerics to '_', so distinct directories ("a/b" and "a_b")
	// can collide on one key — which would collapse them to a single tarball and make
	// both tasks silently ship the same code. Detect that and error instead.
	keyDir := make(map[string]string, len(sources))
	for _, cs := range sources {
		relDir := strings.TrimPrefix(filepath.ToSlash(cs.value), "./")
		key := artifactKey(relDir)
		if prev, ok := keyDir[key]; ok && prev != relDir {
			return diags.Extend(diag.Errorf("code_source directories %q and %q map to the same artifact name %q; rename one so they differ by more than a non-alphanumeric character", prev, relDir, key))
		}
		keyDir[key] = relDir
		outRel := path.Join(codeArtifactOutputDir, key+".tar.gz")
		// Paths are set absolute: a synthesized artifact carries no config location, so
		// artifacts.Prepare cannot resolve relative paths against the bundle root for it.
		// The runtime extracts to /databricks/code_source/<dir>, so entries must nest
		// under the directory basename — hence path = the directory's parent and
		// include = its basename, so the tgz builder names entries "<basename>/...".
		artifacts[key] = &config.Artifact{
			Type:    config.ArtifactTarball,
			Path:    filepath.Join(b.SyncRootPath, filepath.FromSlash(path.Dir(relDir))),
			Include: []string{path.Base(relDir)},
			Files:   []config.ArtifactFile{{Source: filepath.Join(b.SyncRootPath, filepath.FromSlash(outRel))}},
		}
		// code_source_path resolves (via the sync root) to the same absolute output, so
		// libraries upload links the two and rewrites both to the same remote path.
		outputs[cs.configPath.String()] = "./" + outRel
		log.Debugf(ctx, "synthesized tgz artifact %q for code_source_path %q", key, cs.value)
	}

	// Rewrite code_source_path first (via the dynamic tree); the typed Artifacts set
	// below then survives to the dynamic tree on mutator exit. Doing it in the other
	// order would let Mutate's ToTyped pass drop the freshly-set artifacts.
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for _, cs := range sources {
			out := outputs[cs.configPath.String()]
			var err error
			root, err = dyn.SetByPath(root, cs.configPath, dyn.NewValue(out, []dyn.Location{cs.location}))
			if err != nil {
				return root, fmt.Errorf("failed to update code_source_path %q to %q: %w", cs.value, out, err)
			}
		}
		return root, nil
	})
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	if b.Config.Artifacts == nil {
		b.Config.Artifacts = make(map[string]*config.Artifact, len(artifacts))
	}
	maps.Copy(b.Config.Artifacts, artifacts)

	return diags
}

// artifactKey is a stable artifact name for a code directory (relative to the
// bundle). Two tasks pointing at the same directory collapse to one artifact. The
// sanitization is lossy, so distinct directories can collide on one key; the caller
// guards against that (see keyDir in Apply).
func artifactKey(relDir string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, relDir)
	return "air_code_source_" + safe
}

// collectLocalCodeSources returns every AI Runtime task code_source_path that points at
// a local directory. Remote values and local files (handled by the artifact path) are
// skipped.
func collectLocalCodeSources(b *bundle.Bundle) ([]codeSource, diag.Diagnostics) {
	var sources []codeSource
	var diags diag.Diagnostics

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, codeSourcePattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			value, ok := v.AsString()
			if !ok {
				return v, fmt.Errorf("expected string, got %s", v.Kind())
			}
			if !libraries.IsLocalPath(value) {
				return v, nil
			}
			// Only package a local *directory*. A local file (e.g. a pre-built tarball
			// delivered via an `artifacts` block) is left alone so it flows through the
			// standard artifact-upload path as a file.
			localDir := filepath.Join(b.SyncRootPath, filepath.FromSlash(value))
			isDir, err := isExistingDir(localDir)
			if err != nil {
				return v, fmt.Errorf("code_source_path %q: %w", value, err)
			}
			if !isDir {
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

	return sources, diags
}

// isExistingDir reports whether path is an existing directory. A not-exist error is not
// an error here (the path is simply not a directory this mutator packages), but any
// other stat failure — notably a permission error on the parent — is surfaced.
func isExistingDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}
