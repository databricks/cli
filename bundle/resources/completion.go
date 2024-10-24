package resources

import "github.com/databricks/cli/bundle"

// Completions returns the same as [References] except
// that every key maps directly to a single reference.
func Completions(b *bundle.Bundle, filters ...Filter) map[string]Reference {
	out := make(map[string]Reference)
	keyOnlyRefs, _ := References(b, filters...)
	for k, refs := range keyOnlyRefs {
		if len(refs) != 1 {
			continue
		}
		out[k] = refs[0]
	}
	return out
}
