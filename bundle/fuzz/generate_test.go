package fuzz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateJobIsDeterministic(t *testing.T) {
	a := GenerateJob(newRNG(42))
	b := GenerateJob(newRNG(42))
	assert.Equal(t, a, b, "same seed must produce identical job")
}

func TestGenerateJobIsWellFormed(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		job := GenerateJob(newRNG(seed))
		require.NotEmptyf(t, job.Name, "seed %d: job must have a name", seed)
		require.NotEmptyf(t, job.Tasks, "seed %d: job must have at least one task", seed)

		clusterKeys := map[string]bool{}
		for _, jc := range job.JobClusters {
			clusterKeys[jc.JobClusterKey] = true
		}

		taskKeys := map[string]bool{}
		for _, task := range job.Tasks {
			require.NotEmptyf(t, task.TaskKey, "seed %d: task must have a key", seed)
			taskKeys[task.TaskKey] = true

			// A task referencing a job cluster must reference one we generated.
			if task.JobClusterKey != "" {
				assert.Containsf(t, clusterKeys, task.JobClusterKey,
					"seed %d: task %q references unknown job cluster %q", seed, task.TaskKey, task.JobClusterKey)
			}
		}

		// Every dependency must point at a task that exists in this job.
		for _, task := range job.Tasks {
			for _, dep := range task.DependsOn {
				assert.Containsf(t, taskKeys, dep.TaskKey,
					"seed %d: task %q depends on unknown task %q", seed, task.TaskKey, dep.TaskKey)
			}
		}
	}
}
