package run

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"golang.org/x/exp/maps"
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

// ResourceCompletionMap returns a map of resource keys to their respective names.
func ResourceCompletionMap(b *bundle.Bundle) map[string]string {
	out := make(map[string]string)
	keyOnly, keyWithType := ResourceKeys(b)

	// Keep track of resources we have seen by their fully qualified key.
	seen := make(map[string]bool)

	// First add resources that can be identified by key alone.
	for k, v := range keyOnly {
		// Invariant: len(v) >= 1. See [ResourceKeys].
		if len(v) == 1 {
			seen[v[0].Key()] = true
			out[k] = v[0].Name()
		}
	}

	// Then add resources that can only be identified by their type and key.
	for k, v := range keyWithType {
		// Invariant: len(v) == 1. See [ResourceKeys].
		_, ok := seen[v[0].Key()]
		if ok {
			continue
		}
		out[k] = v[0].Name()
	}

	return out
}

// ResourceCompletions returns a list of keys that unambiguously reference resources in the bundle.
func ResourceCompletions(b *bundle.Bundle) []string {
	return maps.Keys(ResourceCompletionMap(b))
}
