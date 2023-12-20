package mutator

import (
	"github.com/databricks/cli/bundle"
)

func DefaultMutators() []bundle.Mutator {
	return []bundle.Mutator{
		// scripts.Execute(config.ScriptPreInit),
		ProcessRootIncludes(),
		RewriteSyncPaths(),
		EnvironmentsToTargets(),
		InitializeVariables(),
		DefineDefaultTarget(),
		LoadGitDetails(),
	}
}

func DefaultMutatorsForTarget(env string) []bundle.Mutator {
	return append(
		DefaultMutators(),
		SelectTarget(env),
		MergeJobClusters(),
		MergeJobTasks(),
		MergePipelineClusters(),
	)
}
