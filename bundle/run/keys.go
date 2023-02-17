package run

import (
	"fmt"

	"github.com/databricks/bricks/bundle"
)

// RunnerLookup maps identifiers to a list of workloads that match that identifier.
// The list can have more than 1 entry if resources of different types use the
// same key. When this happens, the user should disambiguate between them.
type RunnerLookup map[string][]Runner

// ResourceKeys computes a map with
func ResourceKeys(b *bundle.Bundle) (keyOnly RunnerLookup, keyWithType RunnerLookup) {
	keyOnly = make(RunnerLookup)
	keyWithType = make(RunnerLookup)

	// TODO: how do these resources get back the job id?
	r := b.Config.Resources
	for k, v := range r.Jobs {
		kt := fmt.Sprintf("jobs.%s", k)
		w := jobRunner{key: key(kt), bundle: b, job: v}
		keyOnly[k] = append(keyOnly[k], &w)
		keyWithType[kt] = append(keyWithType[kt], &w)
	}
	for k, v := range r.Pipelines {
		kt := fmt.Sprintf("pipelines.%s", k)
		w := pipelineRunner{key: key(kt), bundle: b, pipeline: v}
		keyOnly[k] = append(keyOnly[k], &w)
		keyWithType[kt] = append(keyWithType[kt], &w)
	}
	return
}
