package apps

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"golang.org/x/sync/errgroup"

	"gopkg.in/yaml.v3"
)

type uploadConfig struct {
	filerFactory deploy.FilerFactory
}

func (u *uploadConfig) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	errGroup, ctx := errgroup.WithContext(ctx)

	diagsPerApp := make(map[string]diag.Diagnostic)

	for key, app := range b.Config.Resources.Apps {
		// If the app has a config, we need to deploy it first.
		// It means we need to write app.yml file with the content of the config field
		// to the remote source code path of the app.
		if app.Config != nil {
			if !strings.HasPrefix(app.SourceCodePath, b.Config.Workspace.FilePath) {
				diags = append(diags, diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "App source code invalid",
					Detail:    fmt.Sprintf("App source code path %s is not within file path %s", app.SourceCodePath, b.Config.Workspace.FilePath),
					Locations: b.Config.GetLocations(fmt.Sprintf("resources.apps.%s.source_code_path", key)),
				})

				continue
			}

			appPath := strings.TrimPrefix(app.SourceCodePath, b.Config.Workspace.FilePath)

			buf, err := configToYaml(app)
			if err != nil {
				return diag.FromErr(err)
			}

			// When the app is started, create a new app deployment and wait for it to complete.
			f, err := u.filerFactory(b)
			if err != nil {
				return diag.FromErr(err)
			}

			errGroup.Go(func() error {
				err = f.Write(ctx, path.Join(appPath, "app.yml"), buf, filer.OverwriteIfExists)
				if err != nil {
					diagsPerApp[key] = diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "Failed to save config",
						Detail:    fmt.Sprintf("Failed to write %s file: %s", path.Join(app.SourceCodePath, "app.yml"), err),
						Locations: b.Config.GetLocations(fmt.Sprintf("resources.apps.%s", key)),
					}
				}
				return nil
			})
		}
	}

	if err := errGroup.Wait(); err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	for _, diag := range diagsPerApp {
		diags = append(diags, diag)
	}

	return diags
}

// Name implements bundle.Mutator.
func (u *uploadConfig) Name() string {
	return "apps:UploadConfig"
}

func UploadConfig() bundle.Mutator {
	return &uploadConfig{
		filerFactory: func(b *bundle.Bundle) (filer.Filer, error) {
			return filer.NewWorkspaceFilesClient(b.WorkspaceClient(), b.Config.Workspace.FilePath)
		},
	}
}

func configToYaml(app *resources.App) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)

	err := enc.Encode(app.Config)
	defer enc.Close()

	if err != nil {
		return nil, fmt.Errorf("failed to encode app config to yaml: %w", err)
	}

	return buf, nil
}
