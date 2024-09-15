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
	uploadPath, err := GetUploadBasePath(b)
	if err != nil {
		return diag.FromErr(err)
	}

	// If the client is not initialized, initialize it
	// We use client field in mutator to allow for mocking client in testing
	if u.client == nil {
		filer, diags := GetFilerForLibraries(ctx, b, uploadPath)
		if diags.HasError() {
			return diags
		}

		u.client = filer
	}

	var diags diag.Diagnostics

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

// TODO: TODO: Nicer comments here.
// Case 1: UC volume path is valid. Return the client.
// Case 2: invalid path.
// (a) Not enough elements.
// (b) catalog and schema correspond to a volume define in the DAB.
//
//	-> exception for when the schema value is fully or partially interpolated.
//	   In that case only check the catalog name.
func GetFilerForLibraries(ctx context.Context, b *bundle.Bundle, uploadPath string) (filer.Filer, diag.Diagnostics) {
	w := b.WorkspaceClient()
	isVolumesPath := strings.HasPrefix(uploadPath, "/Volumes/")

	// If the path is not a volume path, use the workspace file system.
	if !isVolumesPath {
		f, err := filer.NewWorkspaceFilesClient(w, uploadPath)
		return f, diag.FromErr(err)
	}

	parts := strings.Split(uploadPath, "/")
	volumeFormatErr := fmt.Errorf("expected UC volume path to be in the format /Volumes/<catalog>/<schema>/<path>, got %s", uploadPath)
	if len(strings.Split(uploadPath, "/")) < 5 {
		return nil, diag.FromErr(volumeFormatErr)
	}

	catalogName := parts[2]
	schemaName := parts[3]
	volumeName := parts[4]

	// Incorrect format.
	if catalogName == "" || schemaName == "" || volumeName == "" {
		return nil, diag.FromErr(volumeFormatErr)
	}

	// If the volume exists already, directly return the filer for the upload path.
	volumePath := fmt.Sprintf("/Volumes/%s/%s/%s", catalogName, schemaName, volumeName)
	vf, err := filer.NewFilesClient(w, volumePath)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	if _, err := vf.Stat(ctx, "."); err == nil {
		f, err := filer.NewFilesClient(w, uploadPath)
		return f, diag.FromErr(err)
	}

	// The volume does not exist. Check if the volume is defined in the bundle.
	// TODO: Does this break? Did it work before if the volume was not defined, but
	// the schema was?
	l, ok := locationForVolume(b, catalogName, schemaName, volumeName)
	if !ok {
		return nil, diag.Errorf("the bundle is configured to upload artifacts to %s but a UC volume at %s does not exist", uploadPath, volumePath)
	}

	return nil, diag.Errorf(`the bundle is configured to upload artifacts to %s but a
UC volume at %s does not exist. Note: We detected that you have a UC volume
defined that matched the path above at %s. Please deploy the UC volume
in a separate deployment before using it in as a destination to upload
artifacts.`, uploadPath, volumePath, l)
}

func locationForVolume(b *bundle.Bundle, catalogName, schemaName, volumeName string) (dyn.Location, bool) {
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
		return b.Config.GetLocation(fmt.Sprintf("resources.volumes.%s", k)), true
	}
	return dyn.Location{}, false
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

func GetUploadBasePath(b *bundle.Bundle) (string, error) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return "", fmt.Errorf("remote artifact path not configured")
	}

	return path.Join(artifactPath, ".internal"), nil
}
