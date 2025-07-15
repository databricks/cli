package libraries

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/utils"

	"golang.org/x/sync/errgroup"
)

// The Files API backend has a rate limit of 10 concurrent
// requests and 100 QPS. We limit the number of concurrent requests to 5 to
// avoid hitting the rate limit.
var maxFilesRequestsInFlight = 5

func Upload() bundle.Mutator {
	return &upload{}
}

func UploadWithClient(client filer.Filer) bundle.Mutator {
	return &upload{
		client: client,
	}
}

type upload struct {
	client filer.Filer
}

type configLocation struct {
	configPath dyn.Path
	location   dyn.Location
}

// Collect all libraries from the bundle configuration and their config paths.
// By this stage all glob references are expanded and we have a list of all libraries that need to be uploaded.
// We collect them from task libraries, foreach task libraries, environment dependencies, and artifacts.
// We return a map of library source to a list of config paths and locations where the library is used.
// We use map so we don't upload the same library multiple times.
// Instead we upload it once and update all the config paths to point to the uploaded location.
func collectLocalLibraries(b *bundle.Bundle) (map[string][]configLocation, error) {
	libs := make(map[string]([]configLocation))

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
				libs[source] = append(libs[source], configLocation{
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

			libs[source] = append(libs[source], configLocation{
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

func (u *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	client, uploadPath, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return diags
	}

	// Only set the filer client if it's not already set. We use the client field
	// in the mutator to mock the filer client in testing
	if u.client == nil {
		u.client = client
	}

	libs, err := collectLocalLibraries(b)
	if err != nil {
		return diag.FromErr(err)
	}

	sources := utils.SortedKeys(libs)

	errs, errCtx := errgroup.WithContext(ctx)
	errs.SetLimit(maxFilesRequestsInFlight)

	for _, source := range sources {
		relPath, err := filepath.Rel(b.SyncRootPath, source)
		if err != nil {
			relPath = source
		} else {
			relPath = filepath.ToSlash(relPath)
		}
		cmdio.LogString(ctx, fmt.Sprintf("Uploading %s...", relPath))
		errs.Go(func() error {
			return UploadFile(errCtx, source, u.client)
		})
	}

	if err := errs.Wait(); err != nil {
		return diag.FromErr(err)
	}

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

	return diags
}

func (u *upload) Name() string {
	return "libraries.Upload"
}

// Function to upload file (a library, artifact and etc) to Workspace or UC volume
func UploadFile(ctx context.Context, file string, client filer.Filer) error {
	filename := filepath.Base(file)

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", file, errors.Unwrap(err))
	}
	defer f.Close()

	err = client.Write(ctx, filename, f, filer.OverwriteIfExists, filer.CreateParentDirectories)
	if err != nil {
		return fmt.Errorf("unable to import %s: %w", filename, err)
	}

	log.Infof(ctx, "Upload succeeded")
	return nil
}
