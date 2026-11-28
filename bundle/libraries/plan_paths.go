package libraries

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/patchwheel"
)

// PlanLocalLibraryPaths returns the local files to upload when applying a saved plan.
// These are the files LocalLibraryPaths returns, plus every other local file that the
// plan's resources reference under the remote upload directory. LocalLibraryPaths
// recomputes the upload set from the local files, while the plan recorded the names
// that existed at "bundle plan" time. The two differ when the files changed in between,
// and for references SwitchToPatchedWheels does not rewrite (such as pipeline
// environment dependencies on a dynamic_version wheel). The upload step empties the
// remote directory first, so a referenced file that is not uploaded is lost. If a
// wheel that a resource the plan changes references has no local counterpart, it
// returns an error, before anything is cleaned up or uploaded.
func PlanLocalLibraryPaths(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) ([]string, error) {
	paths, err := LocalLibraryPaths(ctx, b)
	if err != nil {
		return nil, err
	}

	// Without an upload path there is nothing to compare with; the upload step
	// reports the same error if anything needs uploading.
	_, uploadPath, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return paths, nil
	}

	have := make(map[string]bool, len(paths))
	for _, p := range paths {
		have[filepath.Base(p)] = true
	}
	planned, unchanged := planUploadReferences(plan, uploadPath)
	var missing []string
	for _, name := range slices.Concat(planned, unchanged) {
		if !have[name] {
			missing = append(missing, name)
			have[name] = true
		}
	}
	if len(missing) == 0 {
		return paths, nil
	}

	candidates := localUploadCandidates(ctx, b, missing)
	required := make(map[string]bool, len(planned))
	for _, name := range planned {
		required[name] = true
	}
	var notFound []string
	for _, name := range missing {
		if p, ok := candidates[name]; ok {
			paths = append(paths, p)
			continue
		}
		// Only wheels the plan is about to deploy are required. Unchanged resources
		// can reference files the bundle does not produce (libraries installed on a
		// cluster by a job run, uploads kept by skip_artifact_cleanup), and other
		// strings under the upload path are not necessarily uploads.
		if _, err := patchwheel.ParseWheelFilename(name); err == nil && required[name] {
			notFound = append(notFound, name)
		}
	}
	if len(notFound) > 0 {
		return nil, fmt.Errorf("the plan references %s, which no local file provides; the local files changed since the plan was created or the plan comes from another checkout, run \"bundle plan\" again", strings.Join(notFound, ", "))
	}

	slices.Sort(paths)
	return paths, nil
}

// planUploadReferences returns the sorted base names of the files directly under
// uploadPath that the plan's resources reference: from the planned state of the
// resources the plan changes, and from the remote state of the resources it leaves
// unchanged (their entries carry no planned state), leaving out remote values the plan
// marks as unmanaged (such as libraries a job run installed on a cluster). Resources
// the plan deletes are skipped. A reference can be a whole string or part of one, such as a "%pip install"
// line or a "--wheel=" argument.
func planUploadReferences(plan *deployplan.Plan, uploadPath string) (planned, unchanged []string) {
	plannedNames := make(map[string]bool)
	unchangedNames := make(map[string]bool)
	for _, entry := range plan.Plan {
		if entry == nil || entry.Action == deployplan.Delete {
			continue
		}
		var raw []byte
		names, fromRemote := plannedNames, false
		if entry.NewState != nil && len(entry.NewState.Value) > 0 {
			raw = entry.NewState.Value
		} else if entry.RemoteState != nil {
			b, err := json.Marshal(entry.RemoteState)
			if err != nil {
				continue
			}
			raw = b
			names, fromRemote = unchangedNames, true
		} else {
			continue
		}
		var v any
		if json.Unmarshal(raw, &v) != nil {
			continue
		}
		unmanaged := make(map[string]bool)
		if fromRemote {
			for _, change := range entry.Changes {
				if change == nil || change.Reason != deployplan.ReasonUnmanaged {
					continue
				}
				forEachString(jsonValue(change.Remote), func(s string) {
					for _, name := range uploadNamesIn(s, uploadPath) {
						unmanaged[name] = true
					}
				})
			}
		}
		forEachString(v, func(s string) {
			for _, name := range uploadNamesIn(s, uploadPath) {
				if !unmanaged[name] {
					names[name] = true
				}
			}
		})
	}
	return slices.Sorted(maps.Keys(plannedNames)), slices.Sorted(maps.Keys(unchangedNames))
}

// uploadNamesIn returns the base names of the files directly under uploadPath that s
// mentions, without pip extras ("[train]") or anything after the path.
func uploadNamesIn(s, uploadPath string) []string {
	prefix := uploadPath + "/"
	var names []string
	for {
		i := strings.Index(s, prefix)
		if i < 0 {
			return names
		}
		s = s[i+len(prefix):]
		end := strings.IndexAny(s, " \t\r\n\"'`,;()")
		name := s
		if end >= 0 {
			name = s[:end]
		}
		name, _ = patchwheel.SplitWheelExtras(name)
		if name != "" && !strings.Contains(name, "/") {
			names = append(names, name)
		}
	}
}

// hasPatchedName reports whether one of names could be the patched name of the wheel
// file base, which keeps its distribution and version and appends a local version
// ("pkg-1.0-..." becomes "pkg-1.0+<n>-...").
func hasPatchedName(base string, names []string) bool {
	parts := strings.SplitN(base, "-", 3)
	if len(parts) < 3 {
		return false
	}
	// Patching replaces a local version the wheel already has.
	version, _, _ := strings.Cut(parts[1], "+")
	prefix := parts[0] + "-" + version + "+"
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// jsonValue converts v to the generic form encoding/json decodes into.
func jsonValue(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	return out
}

func forEachString(v any, fn func(string)) {
	switch v := v.(type) {
	case string:
		fn(v)
	case []any:
		for _, x := range v {
			forEachString(x, fn)
		}
	case map[string]any:
		for _, x := range v {
			forEachString(x, fn)
		}
	}
}

// localUploadCandidates maps base names to the local files that could be uploaded:
// every file a library path or glob matches, the patched wheels in the local cache,
// every file an artifact's file glob matches, and the patched copy of each
// dynamic_version wheel whose patched name could be one of names. Unlike
// LocalLibraryPaths it keeps every version, so a plan made before a newer build
// still finds its files.
func localUploadCandidates(ctx context.Context, b *bundle.Bundle, names []string) map[string]string {
	var files []string
	libs, err := collectLocalLibraries(b)
	if err == nil {
		for _, source := range slices.Sorted(maps.Keys(libs)) {
			matches, _ := filepath.Glob(source)
			files = append(files, matches...)
		}
	}

	cacheDir, _ := b.LocalStateDir(ctx)
	for _, artifactName := range slices.Sorted(maps.Keys(b.Config.Artifacts)) {
		// Patched wheels left by "bundle plan" are exactly what it planned to upload.
		if cacheDir != "" {
			cached, _ := filepath.Glob(filepath.Join(cacheDir, "patched_wheels", artifactName+"_*", "*.whl"))
			files = append(files, cached...)
		}
	}
	for _, artifactName := range slices.Sorted(maps.Keys(b.Config.Artifacts)) {
		artifact := b.Config.Artifacts[artifactName]
		if artifact == nil {
			continue
		}
		dynamic := cacheDir != "" && artifact.Type == "whl" && config.IsExplicitlyEnabled(artifact.DynamicVersion)
		for _, f := range artifact.Files {
			sources, _ := filepath.Glob(f.Source)
			files = append(files, sources...)
			if !dynamic {
				continue
			}
			for _, source := range sources {
				info, err := patchwheel.ParseWheelFilename(filepath.Base(source))
				if err != nil || !hasPatchedName(filepath.Base(source), names) {
					continue
				}
				dir := filepath.Join(cacheDir, "patched_wheels", artifactName+"_"+info.Distribution)
				if os.MkdirAll(dir, 0o700) != nil {
					continue
				}
				if patchedPath, _, err := patchwheel.PatchWheel(source, dir); err == nil {
					files = append(files, patchedPath)
				}
			}
		}
	}

	candidates := make(map[string]string, len(files))
	for _, p := range files {
		name := filepath.Base(p)
		if _, ok := candidates[name]; ok {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			candidates[name] = p
		}
	}
	return candidates
}
