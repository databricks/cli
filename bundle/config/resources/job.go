package resources

import (
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/imdario/mergo"
)

type Job struct {
	ID          string       `json:"id,omitempty" bundle:"readonly"`
	Permissions []Permission `json:"permissions,omitempty"`

	Paths

	*jobs.JobSettings
}

// MergeJobClusters merges job clusters with the same key.
// The job clusters field is a slice, and as such, overrides are appended to it.
// We can identify a job cluster by its key, however, so we can use this key
// to figure out which definitions are actually overrides and merge them.
func (j *Job) MergeJobClusters() error {
	keys := make(map[string]*jobs.JobCluster)
	output := make([]jobs.JobCluster, 0, len(j.JobClusters))

	// Environment overrides are always appended, so we can iterate in natural order to
	// first find the base definition, and merge instances we encounter later.
	for i := range j.JobClusters {
		key := j.JobClusters[i].JobClusterKey

		// Register job cluster with key if not yet seen before.
		ref, ok := keys[key]
		if !ok {
			output = append(output, j.JobClusters[i])
			keys[key] = &j.JobClusters[i]
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
