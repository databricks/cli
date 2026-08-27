package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	pathlib "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	libsync "github.com/databricks/cli/libs/sync"
)

type resolveJobRunFileTriggers struct{}

// ResolveJobRunFileTriggers expands on_file_change globs into trigger state.
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
		if jr == nil || !jr.HasOnFileChange() {
			continue
		}
		out := make(map[string]string)
		for i, t := range jr.Lifecycle.Triggers {
			if t.OnFileChange == nil {
				continue
			}
			path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d].on_file_change", name, i)
			pattern, fingerprint, d := resolveFileTrigger(b, path, *t.OnFileChange, syncable)
			diags = diags.Extend(d)
			if !d.HasError() {
				out[pattern] = fingerprint
			}
		}
		jr.Lifecycle.TriggersState = &resources.JobRunTriggersState{OnFileChange: out}
	}
	return diags
}

// syncableRelPaths lists the relative paths sync would upload.
func syncableRelPaths(ctx context.Context, b *bundle.Bundle) ([]string, diag.Diagnostics) {
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

func listSyncableRelPaths(ctx context.Context, b *bundle.Bundle) ([]string, error) {
	fl, err := libsync.NewFileList(ctx, b.WorktreeRoot, b.SyncRoot, b.Config.Sync.Paths, b.Config.Sync.Include, b.Config.Sync.Exclude)
	if err != nil {
		return nil, err
	}
	files, err := fl.Files(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, filepath.ToSlash(f.Relative))
	}
	slices.Sort(out)
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

func resolveFileTrigger(b *bundle.Bundle, loc, pattern string, syncable []string) (string, string, diag.Diagnostics) {
	var diags diag.Diagnostics
	// A double star looks recursive but path.Match treats it as two ordinary stars.
	if strings.Contains(pattern, "**") {
		return "", "", diags.Append(fileTriggerDiag(b, loc, diag.Error, "** in %q is not supported; use * for a single directory level", pattern))
	}
	// A POSIX path is absolute on Windows too, so check both flavours like NormalizePaths does.
	if filepath.IsAbs(pattern) || pathlib.IsAbs(pattern) {
		return "", "", diags.Append(fileTriggerDiag(b, loc, diag.Error, "pattern %q must be relative to the defining YAML file", pattern))
	}
	// NormalizePaths has already rewritten YAML-relative globs to be bundle-root
	// relative. Join that onto the bundle root, then require the result stay
	// under the sync root (an ancestor of the bundle when sync.paths uses ..).
	joined := filepath.Join(b.BundleRootPath, filepath.FromSlash(pattern))
	relPattern, err := filepath.Rel(b.SyncRootPath, joined)
	if err != nil || !filepath.IsLocal(relPattern) {
		return "", "", diags.Append(fileTriggerDiag(b, loc, diag.Error, "pattern %q is not under the sync root", pattern))
	}
	relPattern = filepath.ToSlash(relPattern)
	_, err = pathlib.Match(relPattern, "")
	if err != nil {
		return "", "", diags.Append(fileTriggerDiag(b, loc, diag.Error, "invalid pattern %q: %s", pattern, err))
	}
	h := sha256.New()
	matches := 0
	for _, rel := range syncable {
		matched, err := pathlib.Match(relPattern, rel)
		if err != nil {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Error, "invalid pattern %q: %s", pattern, err))
			continue
		}
		if !matched {
			continue
		}
		match := filepath.Join(b.SyncRootPath, filepath.FromSlash(rel))
		hash, err := hashFile(match)
		if err != nil {
			diags = diags.Append(fileTriggerDiag(b, loc, diag.Error, "hash %q: %s", match, err))
			continue
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write([]byte(hash))
		h.Write([]byte{0})
		matches++
	}
	if matches == 0 && !diags.HasError() {
		diags = diags.Append(fileTriggerDiag(b, loc, diag.Warning, "no synced files match %q", pattern))
	}
	return relPattern, hex.EncodeToString(h.Sum(nil)), diags
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
