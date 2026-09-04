package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	pathlib "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	libsync "github.com/databricks/cli/libs/sync"
)

// Every diagnostic below is reported against an on_file_change entry.
const fileTriggerPrefix = "lifecycle.triggers.on_file_change: "

type resolveJobRunFileTriggers struct{}

// ResolveJobRunFileTriggers expands on_file_change globs into trigger state.
func ResolveJobRunFileTriggers() bundle.Mutator {
	return &resolveJobRunFileTriggers{}
}

func (*resolveJobRunFileTriggers) Name() string {
	return "ResolveJobRunFileTriggers"
}

func (*resolveJobRunFileTriggers) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Sorted so diagnostics from several job_runs come out in a stable order.
	names := make([]string, 0, len(b.Config.Resources.JobRuns))
	for name, jr := range b.Config.Resources.JobRuns {
		// A job_run declared with an empty YAML body is a nil entry here.
		if jr != nil && jr.HasOnFileChange() {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return diags
	}
	slices.Sort(names)

	// Listing the sync files walks the tree, so only do it once the loop above
	// found a pattern that needs matching against it.
	syncable, err := listSyncableRelPaths(ctx, b)
	if err != nil {
		return diags.Append(diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fileTriggerPrefix + fmt.Sprintf("list sync files: %s", err),
		})
	}

	for _, name := range names {
		jr := b.Config.Resources.JobRuns[name]
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

func listSyncableRelPaths(ctx context.Context, b *bundle.Bundle) ([]string, error) {
	// Match sync's effective include set, not just Sync.Include, so a pattern can
	// hash the internal and AI-snapshot dirs sync force-includes.
	includes, err := b.GetSyncIncludePatterns(ctx)
	if err != nil {
		return nil, err
	}
	fl, err := libsync.NewFileList(ctx, b.WorktreeRoot, b.SyncRoot, b.Config.Sync.Paths, includes, b.Config.Sync.Exclude)
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

func resolveFileTrigger(b *bundle.Bundle, loc, pattern string, syncable []string) (string, string, diag.Diagnostics) {
	relPattern, diags := validateFileTriggerPattern(b, loc, pattern)
	if diags.HasError() {
		return "", "", diags
	}

	h := sha256.New()
	matches := 0
	for _, rel := range syncable {
		matched, err := pathlib.Match(relPattern, rel)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fileTriggerPrefix + fmt.Sprintf("invalid pattern %q: %s", pattern, err),
				Locations: b.Config.GetLocations(loc),
			})
			continue
		}
		if !matched {
			continue
		}
		hash, err := hashFile(b.SyncRoot, rel)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fileTriggerPrefix + fmt.Sprintf("hash %q: %s", rel, err),
				Locations: b.Config.GetLocations(loc),
			})
			continue
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write([]byte(hash))
		h.Write([]byte{0})
		matches++
	}
	if matches == 0 && !diags.HasError() {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Warning,
			Summary:   fileTriggerPrefix + fmt.Sprintf("no synced files match %q", pattern),
			Locations: b.Config.GetLocations(loc),
		})
	}
	return relPattern, hex.EncodeToString(h.Sum(nil)), diags
}

func validateFileTriggerPattern(b *bundle.Bundle, loc, pattern string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	// A double star looks recursive but path.Match treats it as two ordinary stars.
	if strings.Contains(pattern, "**") {
		return "", diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fileTriggerPrefix + fmt.Sprintf("** in %q is not supported; use * for a single directory level", pattern),
			Locations: b.Config.GetLocations(loc),
		})
	}
	// filepath.IsAbs only recognises the host's flavour, so check both plus the
	// Windows forms lexically. Otherwise "C:\watched.txt" is rejected on Windows
	// but silently treated as a relative "C:" directory elsewhere, and the same
	// bundle validates differently depending on where it is deployed from.
	if filepath.IsAbs(pattern) || pathlib.IsAbs(pattern) || isWindowsAbs(pattern) {
		return "", diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fileTriggerPrefix + fmt.Sprintf("pattern %q must be relative to the defining YAML file", pattern),
			Locations: b.Config.GetLocations(loc),
		})
	}
	// NormalizePaths has already rewritten YAML-relative globs to be bundle-root
	// relative. Join that onto the bundle root, then require the result stay
	// under the sync root (an ancestor of the bundle when sync.paths uses ..).
	joined := filepath.Join(b.BundleRootPath, filepath.FromSlash(pattern))
	relPattern, err := filepath.Rel(b.SyncRootPath, joined)
	if err != nil || !filepath.IsLocal(relPattern) {
		return "", diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fileTriggerPrefix + fmt.Sprintf("pattern %q is not under the sync root", pattern),
			Locations: b.Config.GetLocations(loc),
		})
	}
	relPattern = filepath.ToSlash(relPattern)
	_, err = pathlib.Match(relPattern, "")
	if err != nil {
		return "", diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fileTriggerPrefix + fmt.Sprintf("invalid pattern %q: %s", pattern, err),
			Locations: b.Config.GetLocations(loc),
		})
	}
	return relPattern, diags
}

// isWindowsAbs reports whether pattern is rooted in Windows terms - a drive
// letter, a UNC share, or a leading separator - whatever the host OS is.
func isWindowsAbs(pattern string) bool {
	if strings.HasPrefix(pattern, `\`) {
		return true
	}
	// "C:", "C:/x" and "C:\x", plus the drive-relative "C:x".
	if len(pattern) < 2 || pattern[1] != ':' {
		return false
	}
	c := pattern[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func hashFile(root fs.FS, path string) (string, error) {
	f, err := root.Open(path)
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
