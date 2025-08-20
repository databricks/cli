package libraries

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/utils"
)

// ReplaceWithRemotePath updates all the libraries paths to point to the remote location
// where the libraries will be uploaded later.
func ReplaceWithRemotePath(ctx context.Context, b *bundle.Bundle) (map[string][]ConfigLocation, diag.Diagnostics) {
	_, uploadPath, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return nil, diags
	}

	libs, err := collectLocalLibraries(b)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	sources := utils.SortedKeys(libs)

	// Update all the config paths to point to the uploaded location
	for _, source := range sources {
		locations := libs[source]
		err = b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			remotePath := path.Join(uploadPath, filepath.Base(source))

			// If the remote path does not start with /Workspace or /Volumes, prepend /Workspace
			if !strings.HasPrefix(remotePath, "/Workspace") && !strings.HasPrefix(remotePath, "/Volumes") {
				remotePath = "/Workspace" + remotePath
			}
			for _, location := range locations {
				v, err = dyn.SetByPath(v, location.configPath, dyn.NewValue(remotePath, []dyn.Location{location.location}))
				if err != nil {
					return v, err
				}
			}

			return v, nil
		})
		if err != nil {
			diags = diags.Extend(diag.FromErr(err))
		}
	}

	return libs, diags
}

// Collect all libraries from the bundle configuration and their config paths.
// By this stage all glob references are expanded and we have a list of all libraries that need to be uploaded.
// We collect them from task libraries, foreach task libraries, environment dependencies, and artifacts.
// We return a map of library source to a list of config paths and locations where the library is used.
// We use map so we don't upload the same library multiple times.
// Instead we upload it once and update all the config paths to point to the uploaded location.
func collectLocalLibraries(b *bundle.Bundle) (map[string][]ConfigLocation, error) {
	libs := make(map[string]([]ConfigLocation))

	patterns := []dyn.Pattern{
		taskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("whl")),
		taskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("jar")),
		forEachTaskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("whl")),
		forEachTaskLibrariesPattern.Append(dyn.AnyIndex(), dyn.Key("jar")),
		envDepsPattern.Append(dyn.AnyIndex()),
		pipelineEnvDepsPattern.Append(dyn.AnyIndex()),
	}

	for _, pattern := range patterns {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				source, ok := v.AsString()
				if !ok {
					return v, fmt.Errorf("expected string, got %s", v.Kind())
				}

				if !IsLibraryLocal(source) {
					return v, nil
				}

				source = filepath.Join(b.SyncRootPath, source)
				libs[source] = append(libs[source], ConfigLocation{
					configPath: p,
					location:   v.Location(),
				})

				return v, nil
			})
		})
		if err != nil {
			return nil, err
		}
	}

	artifactPattern := dyn.NewPattern(
		dyn.Key("artifacts"),
		dyn.AnyKey(),
		dyn.Key("files"),
		dyn.AnyIndex(),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, artifactPattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			file, ok := v.AsMap()
			if !ok {
				return v, fmt.Errorf("expected map, got %s", v.Kind())
			}

			sv, ok := file.GetByString("source")
			if !ok {
				return v, nil
			}

			source, ok := sv.AsString()
			if !ok {
				return v, fmt.Errorf("expected string, got %s", v.Kind())
			}

			if sv, ok = file.GetByString("patched"); ok {
				patched, ok := sv.AsString()
				if ok && patched != "" {
					source = patched
				}
			}

			libs[source] = append(libs[source], ConfigLocation{
				configPath: p.Append(dyn.Key("remote_path")),
				location:   v.Location(),
			})

			return v, nil
		})
	})
	if err != nil {
		return nil, err
	}

	return libs, nil
}
