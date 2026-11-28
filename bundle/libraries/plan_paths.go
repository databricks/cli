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
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/patchwheel"
)

// PlanLocalLibraryPaths returns the local files to upload when applying a saved plan.
// These are the files LocalLibraryPaths returns, plus every other local file that the
// plan's resources reference under the remote upload directory. LocalLibraryPaths
// recomputes the upload set from the local files, while the plan recorded the names
// that existed at "bundle plan" time. The two differ when the files changed in between,
// and for references SwitchToPatchedWheels does not rewrite (such as pipeline
// environment dependencies on a dynamic_version wheel). The upload step empties the
// remote directory first, so a referenced file that is not uploaded is lost. If a file
// the bundle itself uploads for a resource (see planUploadReferences) has no local
// counterpart, it returns an error, before anything is cleaned up or uploaded.
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
	// Artifact cleanup empties the upload directory, so unchanged resources lose their
	// files too unless they are uploaded again.
	skipCleanup := b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.SkipArtifactCleanup)
	required, optional := planUploadReferences(plan, uploadPath, !skipCleanup, newUploadOwners(b))
	var missing []string
	for _, name := range slices.Concat(required, optional) {
		if !have[name] {
			missing = append(missing, name)
			have[name] = true
		}
	}
	if len(missing) == 0 {
		return paths, nil
	}

	candidates := localUploadCandidates(ctx, b, missing)
	isRequired := make(map[string]bool, len(required))
	for _, name := range required {
		isRequired[name] = true
	}
	var notFound []string
	for _, name := range missing {
		if p, ok := candidates[name]; ok {
			paths = append(paths, p)
			continue
		}
		if isRequired[name] {
			notFound = append(notFound, name)
		}
	}
	if len(notFound) > 0 {
		return nil, fmt.Errorf("the plan references %s, which no local file provides; the local files changed since the plan was created or the plan comes from another checkout, run 'bundle plan' again", strings.Join(notFound, ", "))
	}

	slices.Sort(paths)
	return paths, nil
}

// planUploadReferences returns the sorted base names of the files directly under
// uploadPath that the plan's resources reference, split by whether the deploy needs them.
//
// Required are the files the bundle itself uploads for a resource, as owners tells:
//   - in the planned state of a resource the plan creates or updates, a reference
//     anywhere to an artifact's output, or a value that is exactly the uploaded path of
//     one of the resource's own local libraries, at the field where the configuration
//     declares that library;
//   - if requireUnchanged is set, the same library values in the remote state of a
//     resource the plan leaves unchanged (its entry carries no planned state), because
//     artifact cleanup deletes them too.
//
// Optional are all other references, which can name files the bundle does not
// produce: a path the configuration names directly, or the output of a job. Files that
// only the remote side of a change names, such as a library a job run installed on a
// cluster, are left out: plain deploy does not upload them either. Resources the plan
// deletes are skipped.
func planUploadReferences(plan *deployplan.Plan, uploadPath string, requireUnchanged bool, owners uploadOwners) (required, optional []string) {
	requiredNames := make(map[string]bool)
	optionalNames := make(map[string]bool)
	for resourceKey, entry := range plan.Plan {
		if entry == nil || entry.Action == deployplan.Delete {
			continue
		}
		var raw []byte
		fromRemote := false
		if entry.NewState != nil && len(entry.NewState.Value) > 0 {
			raw = entry.NewState.Value
		} else if entry.RemoteState != nil {
			b, err := json.Marshal(entry.RemoteState)
			if err != nil {
				continue
			}
			raw = b
			fromRemote = true
		} else {
			continue
		}
		var v any
		if json.Unmarshal(raw, &v) != nil {
			continue
		}
		var undeclared map[string]bool
		if fromRemote {
			undeclared = remoteOnlyUploadNames(entry.Changes, uploadPath)
		}
		forEachField(v, "", func(key, s string) {
			for _, name := range uploadNamesIn(s, uploadPath) {
				ownLibrary := isUploadPathValue(s, uploadPath, name) && owners.isLibraryOf(resourceKey, key, name)
				switch {
				case undeclared[name]:
				case !fromRemote && (ownLibrary || owners.isArtifactOutput(name)):
					requiredNames[name] = true
				case fromRemote && requireUnchanged && ownLibrary:
					requiredNames[name] = true
				default:
					optionalNames[name] = true
				}
			}
		})
	}
	for name := range requiredNames {
		delete(optionalNames, name)
	}
	return slices.Sorted(maps.Keys(requiredNames)), slices.Sorted(maps.Keys(optionalNames))
}

// remoteOnlyUploadNames returns the base names under uploadPath that the remote value of
// a change names but its configured value does not. In the entry of a resource the plan
// leaves unchanged, these are values the backend has and the configuration does not
// declare, whatever the reason the plan records: "unmanaged" (cluster libraries a job
// run installed when the cluster declares none), "remote_addition" (the same with a
// cluster policy, when the cluster declares libraries of its own), or another reason.
func remoteOnlyUploadNames(changes deployplan.Changes, uploadPath string) map[string]bool {
	names := make(map[string]bool)
	for _, change := range changes {
		if change == nil || change.Remote == nil {
			continue
		}
		declared := make(map[string]bool)
		forEachField(jsonValue(change.New), "", func(_, s string) {
			for _, name := range uploadNamesIn(s, uploadPath) {
				declared[name] = true
			}
		})
		forEachField(jsonValue(change.Remote), "", func(_, s string) {
			for _, name := range uploadNamesIn(s, uploadPath) {
				if !declared[name] {
					names[name] = true
				}
			}
		})
	}
	return names
}

// uploadOwners holds the base name patterns (paths or globs) of the local files the
// bundle uploads: for local libraries per resource ("resources.jobs.my_job") and field
// ("tasks.libraries.whl", without list indices), and for artifact outputs.
type uploadOwners struct {
	libraries map[string]map[string][]string
	artifacts []string
}

func newUploadOwners(b *bundle.Bundle) uploadOwners {
	owners := uploadOwners{libraries: make(map[string]map[string][]string)}
	libs, err := collectLocalLibraries(b)
	if err != nil {
		return owners
	}
	for _, source := range slices.Sorted(maps.Keys(libs)) {
		pattern := filepath.Base(source)
		for _, loc := range libs[source] {
			p := loc.configPath
			switch {
			case len(p) >= 1 && p[0].Key() == "artifacts":
				owners.artifacts = append(owners.artifacts, pattern)
			case len(p) >= 4 && p[0].Key() == "resources":
				key := "resources." + p[1].Key() + "." + p[2].Key()
				if owners.libraries[key] == nil {
					owners.libraries[key] = make(map[string][]string)
				}
				field := fieldPath(p[3:])
				owners.libraries[key][field] = append(owners.libraries[key][field], pattern)
			}
		}
	}
	return owners
}

func (o uploadOwners) isLibraryOf(resourceKey, field, name string) bool {
	return matchesUploadName(o.libraries[resourceKey][field], name)
}

// fieldPath joins the keys of p with dots, leaving out list indices.
func fieldPath(p dyn.Path) string {
	var keys []string
	for _, c := range p {
		if k := c.Key(); k != "" {
			keys = append(keys, k)
		}
	}
	return strings.Join(keys, ".")
}

func (o uploadOwners) isArtifactOutput(name string) bool {
	return matchesUploadName(o.artifacts, name)
}

// matchesUploadName reports whether name is a file one of the patterns matches, or the
// patched copy of one ("pkg-1.0+<n>-..." for "pkg-1.0-...").
func matchesUploadName(patterns []string, name string) bool {
	candidates := []string{name}
	if parts := strings.SplitN(name, "-", 3); len(parts) == 3 && strings.HasSuffix(name, ".whl") {
		if version, _, ok := strings.Cut(parts[1], "+"); ok {
			candidates = append(candidates, parts[0]+"-"+version+"-"+parts[2])
		}
	}
	for _, pattern := range patterns {
		for _, c := range candidates {
			if ok, _ := filepath.Match(pattern, c); ok {
				return true
			}
		}
	}
	return false
}

// isUploadPathValue reports whether s is exactly the uploaded path of name, optionally
// followed by pip extras.
func isUploadPathValue(s, uploadPath, name string) bool {
	base, _ := patchwheel.SplitWheelExtras(s)
	return base == uploadPath+"/"+name
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

// forEachField calls fn for every string in v with its field path: the map keys
// leading to it joined with dots, without list indices.
func forEachField(v any, path string, fn func(path, s string)) {
	switch v := v.(type) {
	case string:
		fn(path, v)
	case []any:
		for _, x := range v {
			forEachField(x, path, fn)
		}
	case map[string]any:
		for k, x := range v {
			next := path + "." + k
			if path == "" {
				next = k
			}
			forEachField(x, next, fn)
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
			cached, _ := filepath.Glob(filepath.Join(patchwheel.CacheDir(cacheDir, artifactName, "*"), "*.whl"))
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
				dir := patchwheel.CacheDir(cacheDir, artifactName, info.Distribution)
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
