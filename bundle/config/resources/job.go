package resources

import (
	"context"
	"strconv"

	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/imdario/mergo"
)

type Job struct {
	ID          string       `json:"id,omitempty" bundle:"readonly"`
	Permissions []Permission `json:"permissions,omitempty"`

	paths.Paths

	*jobs.JobSettings
}

func (s *Job) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Job) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// MergeJobClusters merges job clusters with the same key.
// The job clusters field is a slice, and as such, overrides are appended to it.
// We can identify a job cluster by its key, however, so we can use this key
// to figure out which definitions are actually overrides and merge them.
func (j *Job) MergeJobClusters() error {
	keys := make(map[string]*jobs.JobCluster)
	output := make([]jobs.JobCluster, 0, len(j.JobClusters))

	// Target overrides are always appended, so we can iterate in natural order to
	// first find the base definition, and merge instances we encounter later.
	for i := range j.JobClusters {
		key := j.JobClusters[i].JobClusterKey

		// Register job cluster with key if not yet seen before.
		ref, ok := keys[key]
		if !ok {
			output = append(output, j.JobClusters[i])
			keys[key] = &output[len(output)-1]
			continue
		}

		// Merge this instance into the reference.
		err := mergo.Merge(ref, &j.JobClusters[i], mergo.WithOverride, mergo.WithAppendSlice)
		if err != nil {
			return err
		}
	}

	// Overwrite resulting slice.
	j.JobClusters = output
	return nil
}

// MergeTasks merges tasks with the same key.
// The tasks field is a slice, and as such, overrides are appended to it.
// We can identify a task by its task key, however, so we can use this key
// to figure out which definitions are actually overrides and merge them.
func (j *Job) MergeTasks() error {
	keys := make(map[string]*jobs.Task)
	tasks := make([]jobs.Task, 0, len(j.Tasks))

	// Target overrides are always appended, so we can iterate in natural order to
	// first find the base definition, and merge instances we encounter later.
	for i := range j.Tasks {
		key := j.Tasks[i].TaskKey

		// Register the task with key if not yet seen before.
		ref, ok := keys[key]
		if !ok {
			tasks = append(tasks, j.Tasks[i])
			keys[key] = &tasks[len(tasks)-1]
			continue
		}

		// Merge this instance into the reference.
		err := mergo.Merge(ref, &j.Tasks[i], mergo.WithOverride, mergo.WithAppendSlice)
		if err != nil {
			return err
		}
	}

	// Overwrite resulting slice.
	j.Tasks = tasks
	return nil
}

func (j *Job) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) bool {
	jobId, err := strconv.Atoi(id)
	if err != nil {
		return false
	}
	_, err = w.Jobs.Get(ctx, jobs.GetJobRequest{
		JobId: int64(jobId),
	})
	return err == nil
}

func (j *Job) Type() string {
	return "databricks_job"
}
