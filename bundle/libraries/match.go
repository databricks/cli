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
	for jobKey, job := range b.Config.Resources.Jobs {
		err := validateEnvironments(job.Environments, b, jobKey)
		if err != nil {
			return diag.FromErr(err)
		}

		for i, task := range job.JobSettings.Tasks {
			err := validateTaskLibraries(task.Libraries, b, jobKey, i)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return nil
}

func validateTaskLibraries(libs []compute.Library, b *bundle.Bundle, jobKey string, taskIndex int) error {
	for i, lib := range libs {
		path := libraryPath(&lib)
		if path == "" || !IsLocalPath(path) {
			continue
		}

		loc := b.Config.GetLocation(fmt.Sprintf("resources.jobs.%s.tasks.%d.libraries.%d", jobKey, taskIndex, i))
		matches, err := findMatches(filepath.Join(filepath.Dir(loc.File), path), b)
		if err != nil {
			return err
		}

		if len(matches) == 0 {
			return fmt.Errorf("file %s is referenced in libraries section but doesn't exist on the local file system", libraryPath(&lib))
		}
	}

	return nil
}

func validateEnvironments(envs []jobs.JobEnvironment, b *bundle.Bundle, jobKey string) error {
	for i, env := range envs {
		for j, dep := range env.Spec.Dependencies {
			loc := b.Config.GetLocation(fmt.Sprintf("resources.jobs.%s.environments.%d.spec.dependencies.%d", jobKey, i, j))
			matches, err := findMatches(filepath.Join(filepath.Dir(loc.File), dep), b)
			if err != nil {
				return err
			}

			if len(matches) == 0 && IsLocalPath(dep) {
				return fmt.Errorf("file %s is referenced in environments section but doesn't exist on the local file system", dep)
			}
		}
	}

	return nil
}
