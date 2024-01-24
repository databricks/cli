package dynvar

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"golang.org/x/exp/maps"
)

// Resolve resolves variable references in the given input value using the provided lookup function.
// It returns the resolved output value and any error encountered during the resolution process.
//
// For example, given the input value:
//
//	{
//	    "a": "a",
//	    "b": "${a}",
//	    "c": "${b}${b}",
//	}
//
// The output value will be:
//
//	{
//	    "a": "a",
//	    "b": "a",
//	    "c": "aa",
//	}
//
// If the input value contains a variable reference that cannot be resolved, an error is returned.
// If a cycle is detected in the variable references, an error is returned.
// If for some path the resolution function returns [ErrSkipResolution], the variable reference is left in place.
// This is useful when some variable references are not yet ready to be interpolated.
func Resolve(in dyn.Value, fn Lookup) (out dyn.Value, err error) {
	return resolver{in: in, fn: fn}.run()
}

type resolver struct {
	in dyn.Value
	fn Lookup

	refs     map[string]ref
	resolved map[string]dyn.Value
}

func (r resolver) run() (out dyn.Value, err error) {
	err = r.collectVariableReferences()
	if err != nil {
		return dyn.InvalidValue, err
	}

	err = r.resolveVariableReferences()
	if err != nil {
		return dyn.InvalidValue, err
	}

	out, err = r.replaceVariableReferences()
	if err != nil {
		return dyn.InvalidValue, err
	}

	return out, nil
}

func (r *resolver) collectVariableReferences() (err error) {
	r.refs = make(map[string]ref)

	// First walk the input to gather all values with a variable reference.
	_, err = dyn.Walk(r.in, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		ref, ok := newRef(v, p)
		if !ok {
			// Skip values without variable references.
			return v, nil
		}

		r.refs[ref.key] = ref
		return v, nil
	})

	return err
}

func (r *resolver) resolveVariableReferences() (err error) {
	// Initialize map for resolved variables.
	// We use this for memoization.
	r.resolved = make(map[string]dyn.Value)

	// Resolve each variable reference (in order).
	keys := maps.Keys(r.refs)
	sort.Strings(keys)
	for _, key := range keys {
		_, err := r.resolve(key, []string{key})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *resolver) resolve(key string, seen []string) (dyn.Value, error) {
	// Check if we have already resolved this variable reference.
	if v, ok := r.resolved[key]; ok {
		return v, nil
	}

	ref, ok := r.refs[key]
	if !ok {
		// Perform lookup in the input.
		p, err := dyn.NewPathFromString(key)
		if err != nil {
			return dyn.InvalidValue, err
		}
		v, err := r.fn(p)
		if err != nil && dyn.IsNoSuchKeyError(err) {
			return dyn.InvalidValue, fmt.Errorf(
				"reference does not exist: ${%s}",
				key,
			)
		}
		return v, err
	}

	// This is an unresolved variable reference.
	deps := ref.references()

	// Resolve each of the dependencies, then interpolate them in the ref.
	resolved := make([]dyn.Value, len(deps))
	complete := true

	for j, dep := range deps {
		// Cycle detection.
		if slices.Contains(seen, dep) {
			return dyn.InvalidValue, fmt.Errorf(
				"cycle detected in field resolution: %s",
				strings.Join(append(seen, dep), " -> "),
			)
		}

		v, err := r.resolve(dep, append(seen, dep))

		// If we should skip resolution of this key, index j will hold an invalid [dyn.Value].
		if errors.Is(err, ErrSkipResolution) {
			complete = false
			continue
		} else if err != nil {
			// Otherwise, propagate the error.
			return dyn.InvalidValue, err
		}

		resolved[j] = v
	}

	// Interpolate the resolved values.
	if ref.isPure() && complete {
		// If the variable reference is pure, we can substitute it.
		// This is useful for interpolating values of non-string types.
		r.resolved[key] = resolved[0]
		return resolved[0], nil
	}

	// Not pure; perform string interpolation.
	for j := range ref.matches {
		// The value is invalid if resolution returned [ErrSkipResolution].
		// We must skip those and leave the original variable reference in place.
		if !resolved[j].IsValid() {
			continue
		}

		// Try to turn the resolved value into a string.
		s, ok := resolved[j].AsString()
		if !ok {
			return dyn.InvalidValue, fmt.Errorf(
				"cannot interpolate non-string value: %s",
				ref.matches[j][0],
			)
		}

		ref.str = strings.Replace(ref.str, ref.matches[j][0], s, 1)
	}

	// Store the interpolated value.
	v := dyn.NewValue(ref.str, ref.value.Location())
	r.resolved[key] = v
	return v, nil
}

func (r *resolver) replaceVariableReferences() (dyn.Value, error) {
	// Walk the input and replace all variable references.
	return dyn.Walk(r.in, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		ref, ok := r.refs[p.String()]
		if !ok {
			// No variable reference; return the original value.
			return v, nil
		}

		// We have a variable reference; return the resolved value.
		return r.resolved[ref.key], nil
	})
}
