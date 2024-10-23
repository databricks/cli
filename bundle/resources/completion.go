package resources

import "github.com/databricks/cli/bundle"

// Completions returns the same as [References] except
// that every key maps directly to a single reference.
func Completions(b *bundle.Bundle) map[string]Reference {
	out := make(map[string]Reference)
	for k, refs := range References(b) {
		if len(refs) != 1 {
			continue
		}
		out[k] = refs[0]
	}
	return out
}
