package libraries

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type match struct {
}

func ValidateLocalLibrariesExist() bundle.Mutator {
	return &match{}
}

func (a *match) Name() string {
	return "libraries.ValidateLocalLibrariesExist"
}

func (a *match) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, job := range b.Config.Resources.Jobs {
		err := validateEnvironments(job.Environments, b)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, task := range job.JobSettings.Tasks {
			err := validateTaskLibraries(task.Libraries, b)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return nil
}

func validateTaskLibraries(libs []compute.Library, b *bundle.Bundle) error {
	for _, lib := range libs {
		path := libraryPath(&lib)
		if path == "" || !IsLocalPath(path) {
			continue
		}

		matches, err := filepath.Glob(filepath.Join(b.RootPath, path))
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			return fmt.Errorf("file %s is referenced in libraries section but doesn't exist on the local file system", libraryPath(&lib))
		}
	}

	return nil
}

func validateEnvironments(envs []jobs.JobEnvironment, b *bundle.Bundle) error {
	for _, env := range envs {
		if env.Spec == nil {
			continue
		}

		for _, dep := range env.Spec.Dependencies {
			matches, err := filepath.Glob(filepath.Join(b.RootPath, dep))
			if err != nil {
				return err
			}

			if len(matches) == 0 && IsEnvironmentDependencyLocal(dep) {
				return fmt.Errorf("file %s is referenced in environments section but doesn't exist on the local file system", dep)
			}
		}
	}

	return nil
}
