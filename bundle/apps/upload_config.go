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
	"github.com/databricks/cli/libs/logdiag"
	"golang.org/x/sync/errgroup"

	"gopkg.in/yaml.v3"
)

type uploadConfig struct {
	filerFactory deploy.FilerFactory
}

func (u *uploadConfig) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	errGroup, ctx := errgroup.WithContext(ctx)

	for key, app := range b.Config.Resources.Apps {
		// If the app has a config, we need to deploy it first.
		// It means we need to write app.yml file with the content of the config field
		// to the remote source code path of the app.
		if app.Config != nil {
			appPath := strings.TrimPrefix(app.SourceCodePath, b.Config.Workspace.FilePath)

			buf, err := configToYaml(app)
			if err != nil {
				return diag.FromErr(err)
			}

			f, err := u.filerFactory(b)
			if err != nil {
				return diag.FromErr(err)
			}

			errGroup.Go(func() error {
				err := f.Write(ctx, path.Join(appPath, "app.yml"), buf, filer.OverwriteIfExists)
				if err != nil {
					logdiag.LogDiag(ctx, diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "Failed to save config",
						Detail:    fmt.Sprintf("Failed to write %s file: %s", path.Join(app.SourceCodePath, "app.yml"), err),
						Locations: b.Config.GetLocations("resources.apps." + key),
					})
				}
				return nil
			})
		}
	}

	if err := errGroup.Wait(); err != nil {
		logdiag.LogError(ctx, err)
	}

	return nil
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
