package resourcemutator

import "github.com/databricks/cli/libs/dyn"

// MergeKeyedLists applies the list merges that initialize runs (MergeJobClusters,
// MergeJobParameters, MergeJobTasks, MergePipelineClusters, MergeApps and
// MergeGrants) to v, in the same order. A target that overrides an element of a
// keyed list (e.g. a job task with the same task_key) is concatenated by target
// selection, and merged into one element only by these. It lets a caller see that
// result without running initialize.
func MergeKeyedLists(v dyn.Value) (dyn.Value, error) {
	for _, merge := range []func(dyn.Value) (dyn.Value, error){
		(&mergeJobClusters{}).merge,
		(&mergeJobParameters{}).merge,
		(&mergeJobTasks{}).merge,
		(&mergePipelineClusters{}).merge,
		(&mergeApps{}).merge,
		(&mergeGrants{}).merge,
	} {
		var err error
		v, err = merge(v)
		if err != nil {
			return dyn.InvalidValue, err
		}
	}
	return v, nil
}
