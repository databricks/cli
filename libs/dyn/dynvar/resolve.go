package dynvar

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/utils"
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

type lookupResult struct {
	v   dyn.Value
	err error
}

type resolver struct {
	in dyn.Value
	fn Lookup

	refs     map[string]ref
	resolved map[string]dyn.Value

	// Memoization for lookups.
	lookups map[string]lookupResult
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
		ref, ok := newRef(v)
		if !ok {
			// Skip values without variable references.
			return v, nil
		}

		r.refs[p.String()] = ref
		return v, nil
	})

	return err
}

func (r *resolver) resolveVariableReferences() (err error) {
	// Initialize cache for lookups.
	r.lookups = make(map[string]lookupResult)

	// Initialize cache for resolved variable references.
	r.resolved = make(map[string]dyn.Value)

	// Resolve each variable reference (in order).
	// We sort the keys here to ensure that we always resolve the same variable reference first.
	// This is done such that the cycle detection error is deterministic. If we did not do this,
	// we could enter the cycle at any point in the cycle and return varying errors.
	keys := utils.SortedKeys(r.refs)
	for _, key := range keys {
		v, err := r.resolveRef(r.refs[key], []string{key})
		if err != nil {
			return err
		}
		r.resolved[key] = v
	}

	return nil
}

func (r *resolver) resolveRef(ref ref, seen []string) (dyn.Value, error) {
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

		v, err := r.resolveKey(dep, append(seen, dep))

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
		//
		// Note: we use the location of the variable reference to preserve the information
		// of where it is used. This also means that relative path resolution is done
		// relative to where a variable is used, not where it is defined.
		//
		return dyn.NewValue(resolved[0].Value(), ref.value.Locations()), nil
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

	return dyn.NewValue(ref.str, ref.value.Locations()), nil
}

func (r *resolver) resolveKey(key string, seen []string) (dyn.Value, error) {
	// Check if we have already looked up this key.
	if v, ok := r.lookups[key]; ok {
		return v.v, v.err
	}

	// Parse the key into a path.
	p, err := dyn.NewPathFromString(key)
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Look up the value for the given key.
	v, err := r.fn(p)
	if err != nil {
		if dyn.IsNoSuchKeyError(err) {
			err = fmt.Errorf("reference does not exist: ${%s}", key)
		}

		// Cache the return value and return to the caller.
		r.lookups[key] = lookupResult{v: dyn.InvalidValue, err: err}
		return dyn.InvalidValue, err
	}

	// If the returned value is a valid variable reference, resolve it.
	ref, ok := newRef(v)
	if ok {
		v, err = r.resolveRef(ref, seen)
	}

	// Cache the return value and return to the caller.
	r.lookups[key] = lookupResult{v: v, err: err}
	return v, err
}

func (r *resolver) replaceVariableReferences() (dyn.Value, error) {
	// Walk the input and replace all variable references.
	return dyn.Walk(r.in, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		nv, ok := r.resolved[p.String()]
		if !ok {
			// No variable reference; return the original value.
			return v, nil
		}

		// We have a variable reference; return the resolved value.
		return nv, nil
	})
}
