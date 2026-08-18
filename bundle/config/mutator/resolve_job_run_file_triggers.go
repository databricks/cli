package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
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

func (*resolveJobRunFileTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
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
			hashes, d := resolveFileTrigger(b, path, strings.TrimSpace(*t.OnFileChange))
			diags = diags.Extend(d)
			for k, v := range hashes {
				out[k] = v
			}
		}
		if len(out) == 0 {
			jr.ResolvedFileTriggers = nil
		} else {
			jr.ResolvedFileTriggers = out
		}
	}
	return diags
}

func resolveFileTrigger(b *bundle.Bundle, loc, pattern string) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make(map[string]string)
	localPattern := filepath.FromSlash(pattern)
	// Keep hashes under SyncRoot; same IsLocal gate as translate_paths.
	if !filepath.IsLocal(localPattern) {
		return out, diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: pattern %q is not under the sync root", pattern),
			Locations: b.Config.GetLocations(loc),
		})
	}
	matches, err := filepath.Glob(filepath.Join(b.SyncRootPath, localPattern))
	if err != nil {
		return out, diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: invalid pattern %q: %s", pattern, err),
			Locations: b.Config.GetLocations(loc),
		})
	}
	if len(matches) == 0 {
		out[filepath.ToSlash(pattern)] = missingFileHash
		return out, diags.Append(diag.Diagnostic{
			Severity:  diag.Warning,
			Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: no files match %q", pattern),
			Locations: b.Config.GetLocations(loc),
		})
	}
	regularMatches := 0
	sawNonRegular := false
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: stat %q: %s", match, err),
				Locations: b.Config.GetLocations(loc),
			})
			continue
		}
		if !info.Mode().IsRegular() {
			sawNonRegular = true
			continue
		}
		regularMatches++
		rel, err := filepath.Rel(b.SyncRootPath, match)
		if err != nil || !filepath.IsLocal(rel) {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: matched path %q is not under the sync root", match),
				Locations: b.Config.GetLocations(loc),
			})
			continue
		}
		hash, err := hashFile(match)
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: hash %q: %s", match, err),
				Locations: b.Config.GetLocations(loc),
			})
			continue
		}
		out[filepath.ToSlash(rel)] = hash
	}
	// A directory-only match would otherwise leave ResolvedFileTriggers empty
	// and silently disarm the trigger while config still sets on_file_change.
	if regularMatches == 0 && sawNonRegular {
		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: pattern %q matches no regular files", pattern),
			Locations: b.Config.GetLocations(loc),
		})
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
