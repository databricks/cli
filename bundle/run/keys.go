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

// ResourceCompletions returns a list of keys that unambiguously reference resources in the bundle.
func ResourceCompletions(b *bundle.Bundle) []string {
	seen := make(map[string]bool)
	comps := []string{}
	keyOnly, keyWithType := ResourceKeys(b)

	// First add resources that can be identified by key alone.
	for k, v := range keyOnly {
		// Invariant: len(v) >= 1. See [ResourceKeys].
		if len(v) == 1 {
			seen[v[0].Key()] = true
			comps = append(comps, k)
		}
	}

	// Then add resources that can only be identified by their type and key.
	for k, v := range keyWithType {
		// Invariant: len(v) == 1. See [ResourceKeys].
		_, ok := seen[v[0].Key()]
		if ok {
			continue
		}
		comps = append(comps, k)
	}

	return comps
}
