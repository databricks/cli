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
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"

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
	// If the client is not initialized, initialize it
	// We use client field in mutator to allow for mocking client in testing
	client, uploadPath, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return diags
	}

	// Only set the filer client if it's not already set. This allows for using
	// a mock client in tests.
	if u.client == nil {
		u.client = client
	}

	libs, err := collectLocalLibraries(b)
	if err != nil {
		return diag.FromErr(err)
	}

	errs, errCtx := errgroup.WithContext(ctx)
	errs.SetLimit(maxFilesRequestsInFlight)

	for source := range libs {
		errs.Go(func() error {
			return UploadFile(errCtx, source, u.client)
		})
	}

	if err := errs.Wait(); err != nil {
		return diag.FromErr(err)
	}

	// Update all the config paths to point to the uploaded location
	for source, locations := range libs {
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

// TODO: As a followup add testing for interactive flows.
// This function returns the right filer to use, to upload artifacts to the configured locations.
// Supported locations:
// 1. WSFS
// 2. UC volumes
//
// If a UC Volume is configured, this function checks if the UC volume exists in the workspace.
// Then:
//  1. If the UC volume existing in the workspace:
//     Returns a filer for the UC volume.
//  2. If the UC volume does not exist in the workspace but is very likely to be defined in
//     the bundle configuration:
//     Returns a warning along with the error that instructs the user to deploy the
//     UC volume before using it in the artifact path.
//  3. If the UC volume does not exist in the workspace and is not defined in the bundle configuration:
//     Returns an error.
func GetFilerForLibraries(ctx context.Context, b *bundle.Bundle) (filer.Filer, string, diag.Diagnostics) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return nil, "", diag.Errorf("remote artifact path not configured")
	}

	// path to upload artifact files to.
	uploadPath := path.Join(artifactPath, ".internal")

	w := b.WorkspaceClient()
	isVolumesPath := strings.HasPrefix(uploadPath, "/Volumes/")

	// Return early with a WSFS filer if the artifact path is not a UC volume path.
	if !isVolumesPath {
		f, err := filer.NewWorkspaceFilesClient(w, uploadPath)
		return f, uploadPath, diag.FromErr(err)
	}

	parts := strings.Split(artifactPath, "/")
	volumeFormatErr := fmt.Errorf("expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<path>, got %s", uploadPath)

	// Incorrect format.
	if len(parts) < 5 {
		return nil, "", diag.FromErr(volumeFormatErr)
	}

	catalogName := parts[2]
	schemaName := parts[3]
	volumeName := parts[4]

	// Incorrect format.
	if catalogName == "" || schemaName == "" || volumeName == "" {
		return nil, "", diag.FromErr(volumeFormatErr)
	}

	// Check if the UC volume exists in the workspace.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", catalogName, schemaName, volumeName)
	err := w.Files.GetDirectoryMetadataByDirectoryPath(ctx, volumePath)

	// If the volume exists already, directly return the filer for the upload path.
	if err == nil {
		f, err := filer.NewFilesClient(w, uploadPath)
		return f, uploadPath, diag.FromErr(err)
	}

	diags := diag.Errorf("failed to fetch metadata for the UC volume %s that is configured in the artifact_path: %s", volumePath, err)

	path, locations, ok := matchVolumeInBundle(b, catalogName, schemaName, volumeName)
	if !ok {
		return nil, "", diags
	}

	warning := diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   `the UC volume that is likely being used in the artifact_path has not been deployed yet. Please deploy the UC volume in a separate bundle deploy before using it in the artifact_path.`,
		Locations: locations,
		Paths:     []dyn.Path{path},
	}
	return nil, "", diags.Append(warning)
}

func matchVolumeInBundle(b *bundle.Bundle, catalogName, schemaName, volumeName string) (dyn.Path, []dyn.Location, bool) {
	volumes := b.Config.Resources.Volumes
	for k, v := range volumes {
		if v.CatalogName != catalogName || v.Name != volumeName {
			continue
		}
		// UC schemas can be defined in the bundle itself, and thus might be interpolated
		// at runtime via the ${resources.schemas.<name>} syntax. Thus we match the volume
		// definition if the schema name is the same as the one in the bundle, or if the
		// schema name is interpolated.
		if v.SchemaName != schemaName && !dynvar.ContainsVariableReference(v.SchemaName) {
			continue
		}
		pathString := fmt.Sprintf("resources.volumes.%s", k)
		return dyn.MustPathFromString(pathString), b.Config.GetLocations(pathString), true
	}
	return nil, nil, false
}

// Function to upload file (a library, artifact and etc) to Workspace or UC volume
func UploadFile(ctx context.Context, file string, client filer.Filer) error {
	filename := filepath.Base(file)
	cmdio.LogString(ctx, fmt.Sprintf("Uploading %s...", filename))

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
