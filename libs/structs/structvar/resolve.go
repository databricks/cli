package structvar

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// ErrSkipResolution is returned by a [Lookup] to leave a reference in place.
var ErrSkipResolution = errors.New("skip resolution")

// ReferenceError is returned for a reference to a path that does not exist.
// Suggestions are carried as data so callers (which can import libs/diag) format them.
type ReferenceError struct {
	Reference   string   // original reference text, e.g. "var.hst"
	Suggestions []string // corrected references, e.g. ["var.host", "var.hosts"]
}

func (e *ReferenceError) Error() string {
	return fmt.Sprintf("reference does not exist: ${%s}", e.Reference)
}

// Template is a string with variable references and the locations it was written at.
type Template struct {
	Value     string
	Locations []diag.Location
}

// Lookup returns the value a reference path refers to. It returns
// ErrSkipResolution to leave the reference in place.
type Lookup func(path *structpath.PathNode) (View, error)

// Resolve resolves the references in templates (keyed by path; the keys appear in
// cycle errors) using lookup, with the semantics of dynvar.Resolve: a referenced value
// that is a reference itself is resolved first, cycles are an error, a pure reference
// is replaced by the referenced value (with the location of the reference), and other
// references are interpolated into the string. It returns the views of the resolved
// values; templates that resolve to themselves are omitted.
func Resolve(templates map[string]Template, lookup Lookup) (map[string]View, error) {
	r := resolver{lookup: lookup, lookups: map[string]lookupResult{}}
	out := map[string]View{}

	// Resolve in key order, so the cycle detected is deterministic.
	for _, key := range slices.Sorted(maps.Keys(templates)) {
		t := templates[key]
		ref, ok := NewRef(t.Value)
		if !ok {
			continue
		}
		v, err := r.resolveRef(ref, t.Locations, []string{key})
		if err != nil {
			return nil, err
		}
		if s, ok := v.AsString(); ok && s == t.Value {
			continue
		}
		out[key] = v
	}
	return out, nil
}

type lookupResult struct {
	v   View
	err error
}

type resolver struct {
	lookup  Lookup
	lookups map[string]lookupResult
}

// stringView returns a view of the string s with locations locs.
func stringView(s string, locs []diag.Location) View {
	return NewView(&s, nil, (*Locations)(nil).WithLocations(locs))
}

func (r *resolver) resolveRef(ref Ref, locs []diag.Location, seen []string) (View, error) {
	deps := ref.References()

	// Resolve each of the dependencies, then interpolate them in the ref.
	resolved := make([]View, len(deps))
	complete := true

	for j, dep := range deps {
		// Cycle detection.
		if slices.Contains(seen, dep) {
			return View{}, fmt.Errorf(
				"cycle detected in field resolution: %s",
				strings.Join(append(seen, dep), " -> "),
			)
		}

		v, err := r.resolveKey(dep, append(seen, dep))

		// If we should skip resolution of this key, index j holds an invalid view.
		if errors.Is(err, ErrSkipResolution) {
			complete = false
			continue
		} else if err != nil {
			return View{}, err
		}

		resolved[j] = v
	}

	// A pure reference is replaced by the value, which keeps its type. It takes the
	// location of the reference, so relative paths resolve relative to where a
	// variable is used, not where it is defined.
	if ref.IsPure() && complete {
		return resolved[0].WithLocations(locs), nil
	}

	// Not pure; perform string interpolation. Substitute by byte offset: the same
	// reference may also appear escaped ("$${foo} ${foo}").
	var sb strings.Builder
	consumed := 0
	for j := range ref.Matches {
		// Leave references that were skipped in place.
		if !resolved[j].IsValid() {
			continue
		}

		s, ok := resolved[j].AsString()
		if !ok {
			// Only allow primitive types to be converted to string.
			switch kind := resolved[j].Kind(); kind {
			case KindBool, KindInt, KindFloat, KindNil:
				s = fmt.Sprint(resolved[j].AsAny())
			default:
				return View{}, fmt.Errorf("cannot interpolate non-primitive value of type %s into string", kind)
			}
		}

		start, end := ref.Spans[j][0], ref.Spans[j][1]
		sb.WriteString(ref.Str[consumed:start])
		sb.WriteString(s)
		consumed = end
	}
	sb.WriteString(ref.Str[consumed:])

	return stringView(sb.String(), locs), nil
}

func (r *resolver) resolveKey(key string, seen []string) (View, error) {
	if v, ok := r.lookups[key]; ok {
		return v.v, v.err
	}

	p, err := structpath.ParsePath(key)
	if err != nil {
		return View{}, err
	}

	v, err := r.lookup(p)
	if err != nil {
		if knf, ok := errors.AsType[*KeyNotFoundError](err); ok {
			// Carry suggestions as data; the caller formats them.
			err = &ReferenceError{Reference: key, Suggestions: suggestedReferences(knf, key)}
		}
		r.lookups[key] = lookupResult{err: err}
		return View{}, err
	}

	// If the value is a reference itself, resolve it.
	if s, ok := v.AsString(); ok {
		if ref, ok := NewRef(s); ok {
			v, err = r.resolveRef(ref, v.Locations(), seen)
		}
	}

	r.lookups[key] = lookupResult{v: v, err: err}
	return v, err
}

// WithLocations returns the view with its own locations replaced by locs; the values
// below it keep theirs.
func (x View) WithLocations(locs []diag.Location) View {
	x.loc = x.loc.WithLocations(locs)
	return x
}

// suggestedReferences returns drop-in replacement references for reference, rebuilt
// by swapping the key that was not found for each suggestion.
func suggestedReferences(err *KeyNotFoundError, reference string) []string {
	if len(err.Suggestions) == 0 {
		return nil
	}
	nodes := err.Path.AsSlice()
	failedKey, _ := nodes[len(nodes)-1].StringKey()
	refs := make([]string, len(err.Suggestions))
	for i, s := range err.Suggestions {
		refs[i] = ReplaceKey(reference, failedKey, s)
	}
	return refs
}
