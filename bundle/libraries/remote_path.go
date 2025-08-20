package libraries

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/utils"
)

type remotePath struct{}

// ReplaceWithRemotePath updates all the libraries paths to point to the remote location
// where the libraries will be uploaded later.
func ReplaceWithRemotePath() bundle.Mutator {
	return &remotePath{}
}

func (r *remotePath) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	_, uploadPath, diags := GetFilerForLibraries(ctx, b)
	if diags.HasError() {
		return diags
	}

	libs, err := collectLocalLibraries(b)
	if err != nil {
		return diag.FromErr(err)
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

	return diags
}

func (r *remotePath) Name() string {
	return "libraries.ReplaceWithRemotePath"
}
