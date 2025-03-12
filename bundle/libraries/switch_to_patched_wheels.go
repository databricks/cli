package libraries

import (
	"context"
	"encoding/json"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type switchToPatchedWheels struct{}

func (c switchToPatchedWheels) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	m, _ := json.Marshal(b.Config.Artifacts)
	log.Warnf(ctx, "artifacts: %s", m)

	replacements := getReplacements(b.Config.Artifacts)
	log.Warnf(ctx, "replacements: %v", replacements)

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
			for libInd, libraries := range task.Libraries {
				repl := replacements[libraries.Whl]
				if repl != "" {
					log.Debugf(ctx, "Updating resources.jobs.%s.task[%d].libraries[%d].whl from %s to %s", jobName, taskInd, libInd, libraries.Whl, repl)
					libraries.Whl = repl
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

func getReplacements(artifacts config.Artifacts) map[string]string {
	result := make(map[string]string)
	for _, artifact := range artifacts {
		for _, file := range artifact.Files {
			if file.Patched != "" {
				result[file.Source] = file.Patched
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
