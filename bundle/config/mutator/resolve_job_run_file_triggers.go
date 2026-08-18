package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

// missingFileSentinelSize marks a pattern with no matching file so the next
// plan can distinguish "still missing" from "file appeared".
const missingFileSentinelSize = int64(-1)

type resolveJobRunFileTriggers struct{}

// ResolveJobRunFileTriggers expands on_file_change globs and stores per-file
// fingerprints on each job_run for PrepareState to copy into local state.
func ResolveJobRunFileTriggers() bundle.Mutator {
	return &resolveJobRunFileTriggers{}
}

func (*resolveJobRunFileTriggers) Name() string {
	return "ResolveJobRunFileTriggers"
}

func (*resolveJobRunFileTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for name, jr := range b.Config.Resources.JobRuns {
		if jr == nil {
			continue
		}
		patterns := jr.OnFileChangePatterns()
		if len(patterns) == 0 {
			continue
		}
		fps, d := resolveFileTriggers(b, name, patterns, previousFileTriggers(b, name))
		diags = diags.Extend(d)
		jr.ResolvedFileTriggers = fps
	}
	return diags
}

// previousFileTriggers reads on_file_change fingerprints from deployment state
// when it is open (plan/deploy after StatePull). Used so unchanged content keeps
// a stable fingerprint across mtime-only updates (e.g. touch).
func previousFileTriggers(b *bundle.Bundle, name string) map[string]resources.JobRunFileFingerprint {
	if b.DeploymentBundle.StateDB.Path == "" {
		return nil
	}
	entry, ok := b.DeploymentBundle.StateDB.GetResourceEntry("resources.job_runs." + name)
	if !ok || len(entry.State) == 0 {
		return nil
	}
	var state struct {
		Lifecycle *struct {
			Triggers *struct {
				OnFileChange map[string]resources.JobRunFileFingerprint `json:"on_file_change"`
			} `json:"triggers"`
		} `json:"lifecycle"`
	}
	if err := json.Unmarshal(entry.State, &state); err != nil {
		return nil
	}
	if state.Lifecycle == nil || state.Lifecycle.Triggers == nil {
		return nil
	}
	return state.Lifecycle.Triggers.OnFileChange
}

func resolveFileTriggers(b *bundle.Bundle, name string, patterns []string, prev map[string]resources.JobRunFileFingerprint) (map[string]resources.JobRunFileFingerprint, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make(map[string]resources.JobRunFileFingerprint)
	for _, pattern := range patterns {
		path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers", name)
		localPattern := filepath.FromSlash(pattern)
		// Keep fingerprints under SyncRoot; same IsLocal gate as translate_paths.
		if !filepath.IsLocal(localPattern) {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: pattern %q is not under the sync root", pattern),
				Locations: b.Config.GetLocations(path),
			})
			continue
		}
		matches, err := filepath.Glob(filepath.Join(b.SyncRootPath, localPattern))
		if err != nil {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: invalid pattern %q: %s", pattern, err),
				Locations: b.Config.GetLocations(path),
			})
			continue
		}
		if len(matches) == 0 {
			// Distinct state when the path/glob matches nothing so appear/disappear recreates.
			out[filepath.ToSlash(pattern)] = resources.JobRunFileFingerprint{
				Size: missingFileSentinelSize,
			}
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: no files match %q", pattern),
				Locations: b.Config.GetLocations(path),
			})
			continue
		}
		regularMatches := 0
		sawNonRegular := false
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: stat %q: %s", match, err),
					Locations: b.Config.GetLocations(path),
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
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			key := filepath.ToSlash(rel)
			fp, err := fingerprintFile(match, info, prev[key])
			if err != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: hash %q: %s", match, err),
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			out[key] = fp
		}
		// A directory-only match would otherwise leave ResolvedFileTriggers empty
		// and silently disarm the trigger while config still sets on_file_change.
		if regularMatches == 0 && sawNonRegular {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.triggers.on_file_change: pattern %q matches no regular files", pattern),
				Locations: b.Config.GetLocations(path),
			})
		}
	}
	return out, diags
}

// fingerprintFile builds a content fingerprint. If prev has the same size and
// mtime, it is reused without reading the file. If content hash matches prev,
// prev is reused so mtime-only updates (touch) do not change planned state.
func fingerprintFile(path string, info os.FileInfo, prev resources.JobRunFileFingerprint) (resources.JobRunFileFingerprint, error) {
	size := info.Size()
	mtime := info.ModTime().UnixNano()
	if prev.Hash != "" && prev.Size == size && prev.MtimeNs == mtime {
		return prev, nil
	}
	hash, err := hashFile(path)
	if err != nil {
		return resources.JobRunFileFingerprint{}, err
	}
	if prev.Hash != "" && prev.Hash == hash {
		return prev, nil
	}
	return resources.JobRunFileFingerprint{
		Hash:    hash,
		Size:    size,
		MtimeNs: mtime,
	}, nil
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
