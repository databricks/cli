package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"os"
	pathlib "path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	libsync "github.com/databricks/cli/libs/sync"
)

// missingFileHash marks a pattern with no matching file so appear/disappear recreates.
const missingFileHash = ""

type resolveJobRunFileTriggers struct{}

// ResolveJobRunFileTriggers expands on_file_change globs and stores per-file
// content hashes on each job_run for PrepareState to copy into local state.
func ResolveJobRunFileTriggers() bundle.Mutator {
	return &resolveJobRunFileTriggers{}
}

func (*resolveJobRunFileTriggers) Name() string {
	return "ResolveJobRunFileTriggers"
}

func (*resolveJobRunFileTriggers) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	syncable, diags := syncableRelPaths(ctx, b)
	if diags.HasError() {
		return diags
	}
	for name, jr := range b.Config.Resources.JobRuns {
		if jr == nil || jr.Lifecycle == nil {
			continue
		}
		out := make(map[string]string)
		for i, t := range jr.Lifecycle.Triggers {
			if t.OnFileChange == nil {
				continue
			}
			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d].on_file_change", name, i)
			hashes, d := resolveFileTrigger(b, path, *t.OnFileChange, syncable)
			diags = diags.Extend(d)
			maps.Copy(out, hashes)
		}
		jr.ResolvedFileTriggers = out
	}
	return diags
}

// syncableRelPaths is the set of relative paths sync would upload.
func syncableRelPaths(ctx context.Context, b *bundle.Bundle) (map[string]struct{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	needs := false
	for _, jr := range b.Config.Resources.JobRuns {
		if jr != nil && jr.HasOnFileChange() {
			needs = true
			break
		}
	}
	if !needs {
		return nil, diags
	}

	out, err := listSyncableRelPaths(ctx, b)
	if err != nil {
		return nil, diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("lifecycle.triggers.on_file_change: list sync files: %s", err),
		})
	}
	return out, diags
}

func listSyncableRelPaths(ctx context.Context, b *bundle.Bundle) (map[string]struct{}, error) {
	fl, err := libsync.NewFileList(ctx, b.WorktreeRoot, b.SyncRoot, b.Config.Sync.Paths, b.Config.Sync.Include, b.Config.Sync.Exclude)
	if err != nil {
		return nil, err
	}
	files, err := fl.Files(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(files))
	for _, f := range files {
		out[filepath.ToSlash(f.Relative)] = struct{}{}
	}
	return out, nil
}

// fileTriggerDiag reports a diagnostic against the on_file_change entry at loc.
func fileTriggerDiag(b *bundle.Bundle, loc string, severity diag.Severity, format string, args ...any) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  severity,
		Summary:   "lifecycle.triggers.on_file_change: " + fmt.Sprintf(format, args...),
		Locations: b.Config.GetLocations(loc),
	}
}

func resolveFileTrigger(b *bundle.Bundle, loc, pattern string, syncable map[string]struct{}) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make(map[string]string)
	// filepath.Glob treats ** as two *, so doublestar-style patterns match less than expected.
	if strings.Contains(pattern, "**") {
		return out, diags.Append(fileTriggerDiag(b, loc, diag.Error, "** in %q is not supported; use * for a single directory level", pattern))
	}
	// filepath.Join would otherwise rebase an absolute pattern under the bundle
	// root (Join("/bundle", "/etc/passwd") is "/bundle/etc/passwd"). A POSIX path
	// is absolute on Windows too, so check both flavours like NormalizePaths does.
	if filepath.IsAbs(pattern) || pathlib.IsAbs(pattern) {
		return out, diags.Append(fileTriggerDiag(b, loc, diag.Error, "pattern %q must be relative to the defining YAML file", pattern))
	}
	// NormalizePaths has already rewritten YAML-relative globs to be bundle-root
	// relative. Join that onto the bundle root, then require the result stay
	// under the sync root (an ancestor of the bundle when sync.paths uses ..).
	joined := filepath.Join(b.BundleRootPath, filepath.FromSlash(pattern))
	relPattern, err := filepath.Rel(b.SyncRootPath, joined)
	if err != nil || !filepath.IsLocal(relPattern) {
		return out, diags.Append(fileTriggerDiag(b, loc, diag.Error, "pattern %q is not under the sync root", pattern))
	}
	matches, err := filepath.Glob(joined)
	if err != nil {
		return out, diags.Append(fileTriggerDiag(b, loc, diag.Error, "invalid pattern %q: %s", pattern, err))
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Error, "stat %q: %s", match, err))
			continue
		}
		// A glob like migrations/* routinely matches subdirectories; there is
		// nothing to hash and nothing for the user to fix, so skip them quietly.
		if !info.Mode().IsRegular() {
			continue
		}
		rel, err := filepath.Rel(b.SyncRootPath, match)
		if err != nil || !filepath.IsLocal(rel) {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Error, "matched path %q is not under the sync root", match))
			continue
		}
		// Honor .gitignore and sync.exclude the same way sync does.
		if _, ok := syncable[filepath.ToSlash(rel)]; !ok {
			continue
		}
		hash, err := hashFile(match)
		if err != nil {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Error, "hash %q: %s", match, err))
			continue
		}
		out[filepath.ToSlash(rel)] = hash
	}
	// A pattern that hashes nothing is a warning, not an error: every such case
	// re-arms once a matching file appears. Record the placeholder under the
	// pattern's own sync-root-relative key so that appearance is a hash change
	// rather than a key swap. Skip it when a match failed to be read, since the
	// error already says the fingerprint is incomplete.
	if len(out) == 0 && !diags.HasError() {
		out[filepath.ToSlash(relPattern)] = missingFileHash
		if len(matches) == 0 {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Warning, "no files match %q", pattern))
		} else {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Warning, "pattern %q matches only directories or files excluded from sync, so nothing is hashed", pattern))
		}
	}
	return out, diags
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
