package libraries

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type switchToPatchedWheels struct{}

func (c switchToPatchedWheels) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	replacements := getReplacements(ctx, b.Config.Artifacts, b.BundleRoot.Native())

	for jobName, jobRef := range b.Config.Resources.Jobs {
		if jobRef == nil {
			continue
		}

		job := jobRef.JobSettings

		if job == nil {
			continue
		}

		// resources.jobs.*.task[*].libraries[*]

		for taskInd, task := range job.Tasks {
			for libInd, lib := range task.Libraries {
				repl := replacements[lib.Whl]
				if repl != "" {
					log.Debugf(ctx, "Updating resources.jobs.%s.task[%d].libraries[%d].whl from %s to %s", jobName, taskInd, libInd, lib.Whl, repl)
					task.Libraries[libInd].Whl = repl
				}
			}
		}

		// resources.jobs.*.task[*].*.for_each_task.task.libraries
		// TODO

		// resources.jobs.*.environments.*.spec.dependencies
		// TODO
	}

	return nil
}

func getReplacements(ctx context.Context, artifacts config.Artifacts, bundleRoot string) map[string]string {
	result := make(map[string]string)
	for _, artifact := range artifacts {
		for _, file := range artifact.Files {
			if file.Patched != "" {
				source, err := filepath.Rel(bundleRoot, file.Source)
				if err != nil {
					log.Debugf(ctx, "Failed to get relative path (%s, %s): %s", bundleRoot, file.Source, err)
					continue
				}
				patched, err := filepath.Rel(bundleRoot, file.Patched)
				if err != nil {
					log.Debugf(ctx, "Failed to get relative path (%s, %s): %s", bundleRoot, file.Patched, err)
					continue
				}
				result[source] = patched
				// There already was a check for duplicate by same_name_libraries.go
			}
		}
	}
	return result
}

func (c switchToPatchedWheels) Name() string {
	return "SwitchToPatchedWheels"
}

func SwitchToPatchedWheels() bundle.Mutator {
	return switchToPatchedWheels{}
}
