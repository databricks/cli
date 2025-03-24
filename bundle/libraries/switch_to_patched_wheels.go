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

		for taskInd, task := range job.Tasks {

			// Update resources.jobs.*.task[*].libraries[*].whl
			for libInd, lib := range task.Libraries {
				repl := replacements[lib.Whl]
				if repl != "" {
					log.Debugf(ctx, "Updating resources.jobs.%s.task[%d].libraries[%d].whl from %s to %s", jobName, taskInd, libInd, lib.Whl, repl)
					job.Tasks[taskInd].Libraries[libInd].Whl = repl
				}
			}

			// Update resources.jobs.*.task[*].for_each_task.task.libraries[*].whl

			foreachptr := task.ForEachTask
			if foreachptr != nil {
				for libInd, lib := range foreachptr.Task.Libraries {
					repl := replacements[lib.Whl]
					if repl != "" {
						log.Debugf(ctx, "Updating resources.jobs.%s.task[%d].for_each_task.task.libraries[%d].whl from %s to %s", jobName, taskInd, libInd, lib.Whl, repl)
						foreachptr.Task.Libraries[libInd].Whl = repl
					}
				}
			}
		}

		// Update resources.jobs.*.environments.*.spec.dependencies[*]
		for envInd, env := range job.Environments {
			specptr := env.Spec
			if specptr == nil {
				continue
			}
			for depInd, dep := range specptr.Dependencies {
				repl := replacements[dep]
				if repl != "" {
					log.Debugf(ctx, "Updating resources.jobs.%s.environments[%d].spec.dependencies[%d] from %s to %s", jobName, envInd, depInd, dep, repl)
					specptr.Dependencies[depInd] = repl
				}
			}
		}
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
