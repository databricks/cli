package libraries

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/filer"

	"github.com/databricks/databricks-sdk-go"

	"golang.org/x/sync/errgroup"
)

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

type libsLocations = map[string]([]configLocation)

// Collect all libraries from the bundle configuration and their config paths.
// By this stage all glob references are expanded and we have a list of all libraries that need to be uploaded.
// We collect them from task libraries, foreach task libraries  environment dependencies and artifacts.
// We return a map of library source to a list of config paths and locations where the library is used.
// We use map so we don't upload the same library multiple times.
// Instead we upload it once and update all the config paths to point to the uploaded location.
func collectLocalLibraries(b *bundle.Bundle) libsLocations {
	libs := make(map[string]([]configLocation))

	for i, job := range b.Config.Resources.Jobs {
		for j, task := range job.Tasks {
			for k, lib := range task.Libraries {
				if !IsLocalLibrary(&lib) {
					continue
				}

				p := dyn.NewPath(
					dyn.Key("resources"),
					dyn.Key("jobs"),
					dyn.Key(i),
					dyn.Key("tasks"),
					dyn.Index(j),
					dyn.Key("libraries"),
					dyn.Index(k),
				)
				if lib.Whl != "" {
					libs[lib.Whl] = append(libs[lib.Whl], configLocation{
						configPath: p.Append(dyn.Key("whl")),
						location:   b.Config.GetLocation(p.String()),
					})
				}

				if lib.Jar != "" {
					libs[lib.Jar] = append(libs[lib.Jar], configLocation{
						configPath: p.Append(dyn.Key("jar")),
						location:   b.Config.GetLocation(p.String()),
					})
				}
			}

			if task.ForEachTask != nil {
				for l, lib := range task.ForEachTask.Task.Libraries {
					if !IsLocalLibrary(&lib) {
						continue
					}

					p := dyn.NewPath(
						dyn.Key("resources"),
						dyn.Key("jobs"),
						dyn.Key(i),
						dyn.Key("tasks"),
						dyn.Index(j),
						dyn.Key("for_each_task"),
						dyn.Key("task"),
						dyn.Key("libraries"),
						dyn.Index(l),
					)

					if lib.Whl != "" {
						libs[lib.Whl] = append(libs[lib.Whl], configLocation{
							configPath: p.Append(dyn.Key("whl")),
							location:   b.Config.GetLocation(p.String()),
						})
					}

					if lib.Jar != "" {
						libs[lib.Jar] = append(libs[lib.Jar], configLocation{
							configPath: p.Append(dyn.Key("jar")),
							location:   b.Config.GetLocation(p.String()),
						})
					}
				}
			}
		}

		for j, env := range job.Environments {
			if env.Spec == nil {
				continue
			}

			for k, dep := range env.Spec.Dependencies {
				if !IsLibraryLocal(dep) {
					continue
				}

				p := dyn.NewPath(
					dyn.Key("resources"),
					dyn.Key("jobs"),
					dyn.Key(i),
					dyn.Key("environments"),
					dyn.Index(j),
					dyn.Key("spec"),
					dyn.Key("dependencies"),
					dyn.Index(k),
				)

				libs[dep] = append(libs[dep], configLocation{
					configPath: p,
					location:   b.Config.GetLocation(p.String()),
				})
			}
		}
	}

	for key, artifact := range b.Config.Artifacts {
		for i, file := range artifact.Files {
			p := dyn.NewPath(
				dyn.Key("artifacts"),
				dyn.Key(key),
				dyn.Key("files"),
				dyn.Index(i),
			)

			libs[file.Source] = append(libs[file.Source], configLocation{
				configPath: p.Append(dyn.Key("remote_path")),
				location:   b.Config.GetLocation(p.String()),
			})
		}
	}

	return libs
}

func (u *upload) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	uploadPath, err := GetUploadBasePath(b)
	if err != nil {
		return diag.FromErr(err)
	}

	// If the client is not initialized, initialize it
	// We use client field in mutator to allow for mocking client in testing
	if u.client == nil {
		filer, err := GetFilerForLibraries(b.WorkspaceClient(), uploadPath)
		if err != nil {
			return diag.FromErr(err)
		}

		u.client = filer
	}

	var diags diag.Diagnostics

	libs := collectLocalLibraries(b)
	errs, errCtx := errgroup.WithContext(ctx)
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

func GetFilerForLibraries(w *databricks.WorkspaceClient, uploadPath string) (filer.Filer, error) {
	if isVolumesPath(uploadPath) {
		return filer.NewFilesClient(w, uploadPath)
	}
	return filer.NewWorkspaceFilesClient(w, uploadPath)
}

func isVolumesPath(path string) bool {
	return strings.HasPrefix(path, "/Volumes/")
}

// Function to upload file (a library, artifact and etc) to Workspace or UC volume
func UploadFile(ctx context.Context, file string, client filer.Filer) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("unable to read %s: %w", file, errors.Unwrap(err))
	}

	filename := filepath.Base(file)
	err = client.Write(ctx, filename, bytes.NewReader(raw), filer.OverwriteIfExists, filer.CreateParentDirectories)
	if err != nil {
		return fmt.Errorf("unable to import %s: %w", filename, err)
	}

	return nil
}

func GetUploadBasePath(b *bundle.Bundle) (string, error) {
	artifactPath := b.Config.Workspace.ArtifactPath
	if artifactPath == "" {
		return "", fmt.Errorf("remote artifact path not configured")
	}

	return path.Join(artifactPath, ".internal"), nil
}
