package libraries

import (
	"context"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/patchwheel"
	"github.com/databricks/cli/libs/structs/structpath"
)

// LocalLibraryPaths returns the local file paths of all libraries in the bundle
// config that need to be uploaded. It expands any glob patterns that haven't
// been resolved yet (ExpandGlobReferences only runs during Build, not when
// applying a saved plan). For dynamic_version wheel artifacts it computes the
// patched path from the source file's mtime, reusing the cached file that
// "bundle plan" already created rather than re-patching.
func LocalLibraryPaths(ctx context.Context, b *bundle.Bundle) ([]string, error) {
	libs, err := collectLocalLibraries(b)
	if err != nil {
		return nil, err
	}

	// Expand any glob patterns in the collected paths. Job task library globs
	// (e.g. "./dist/*.whl") are expanded by ExpandGlobReferences during Build but
	// not when applying a saved plan, so they may still be present here.
	expanded := make(map[string][]LocationToUpdate, len(libs))
	for source, locs := range libs {
		if strings.ContainsAny(source, "*?[") {
			matches, _ := filepath.Glob(source)
			for _, match := range matches {
				expanded[match] = append(expanded[match], locs...)
			}
		} else {
			expanded[source] = locs
		}
	}
	libs = expanded

	// For dynamic_version wheel artifacts, replace the original source path with
	// the patched path. patchwheel.PatchWheel returns the already-cached result
	// without rebuilding.
	cacheDir, _ := b.LocalStateDir(ctx)
	if cacheDir != "" {
		patches := make(map[string]string) // original → patched
		for artifactName, artifact := range b.Config.Artifacts {
			if artifact == nil || artifact.Type != "whl" || !artifact.DynamicVersion {
				continue
			}
			for _, f := range artifact.Files {
				sources, _ := filepath.Glob(f.Source)
				for _, source := range sources {
					info, err := patchwheel.ParseWheelFilename(filepath.Base(source))
					if err != nil {
						continue
					}
					dir := filepath.Join(cacheDir, "patched_wheels", artifactName+"_"+info.Distribution)
					patchedPath, _, err := patchwheel.PatchWheel(source, dir)
					if err == nil {
						patches[source] = patchedPath
					}
					break
				}
			}
		}
		if len(patches) > 0 {
			patched := make(map[string][]LocationToUpdate, len(libs))
			for k, v := range libs {
				if p, ok := patches[k]; ok {
					patched[p] = append(patched[p], v...)
				} else {
					patched[k] = v
				}
			}
			libs = patched
		}
	}

	return slices.Sorted(maps.Keys(libs)), nil
}

// ReplaceWithRemotePath updates all the libraries paths to point to the remote location
// where the libraries will be uploaded later.
func ReplaceWithRemotePath(ctx context.Context, b *bundle.Bundle) (map[string][]LocationToUpdate, diag.Diagnostics) {
	_, uploadPath, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return nil, diags
	}

	libs, err := collectLocalLibraries(b)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	sources := slices.Sorted(maps.Keys(libs))

	// Update all the config paths to point to the uploaded location
	for _, source := range sources {
		locations := libs[source]
		remotePath := path.Join(uploadPath, filepath.Base(source))

		for _, location := range locations {
			// Re-append the extras suffix that was stripped before upload.
			remotePathWithExtras := remotePath + location.extras
			err = b.Config.Set(location.configPath, remotePathWithExtras)
			if err != nil {
				diags = diags.Extend(diag.FromErr(fmt.Errorf("internal error: failed to update path %#v to %#v: %w", source, remotePathWithExtras, err)))
				return libs, diags
			}
			b.Config.SetLocations(location.configPath, []diag.Location{location.location})
		}
	}

	return libs, diags
}

// Collect all libraries from the bundle configuration and their config paths.
// By this stage all glob references are expanded and we have a list of all libraries that need to be uploaded.
// We collect them from task libraries, foreach task libraries, environment dependencies, and artifacts.
// We return a map of library source to a list of config paths and locations where the library is used.
// We use map so we don't upload the same library multiple times.
// This map is later used to upload the libraries to the remote location and update the config paths to point to the uploaded location.
func collectLocalLibraries(b *bundle.Bundle) (map[string][]LocationToUpdate, error) {
	libs := make(map[string]([]LocationToUpdate))

	// The AI Runtime task's code_source_path is a local archive (typically an
	// artifact-built .tar.gz) that must be uploaded and referenced by its remote path,
	// exactly like a wheel or jar library.
	err := visitLibraryValues(&b.Config.Resources, true, func(p *structpath.PathNode, s *string) error {
		source := *s
		if !IsLibraryLocal(source) {
			return nil
		}

		// Split off any pip extras suffix so the upload targets the real
		// file; the suffix is re-appended to the remote path afterwards.
		source, extras := patchwheel.SplitWheelExtras(source)

		source = filepath.Join(b.SyncRootPath, source)
		libs[source] = append(libs[source], LocationToUpdate{
			configPath: p,
			location:   b.Config.GetLocationOfExact(p),
			extras:     extras,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, name := range slices.Sorted(maps.Keys(b.Config.Artifacts)) {
		a := b.Config.Artifacts[name]
		if a == nil {
			continue
		}
		for i, f := range a.Files {
			p := structpath.NewPath(nil, "artifacts", name, "files", i)
			// An empty source is missing (ArtifactFile has no ForceSendFields).
			if f.Source == "" {
				continue
			}
			source := f.Source
			if f.Patched != "" {
				source = f.Patched
			}
			libs[source] = append(libs[source], LocationToUpdate{
				configPath: structpath.NewStringKey(p, "remote_path"),
				location:   b.Config.GetLocationOfExact(p),
			})
		}
	}

	return libs, nil
}
