package dynvar

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"golang.org/x/exp/maps"
)

var re = regexp.MustCompile(`\$\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*)*)\}`)

type variableReference struct {
	value   dyn.Value
	path    dyn.Path
	str     string
	matches [][]string
}

// isPure returns true if the variable reference contains a single
// variable reference and nothing more. We need this so we can
// interpolate values of non-string types (i.e. it can be substituted).
func (v variableReference) isPure() bool {
	// Need single match, equal to the incoming string.
	if len(v.matches) == 0 || len(v.matches[0]) == 0 {
		panic("invalid variable reference; expect at least one match")
	}
	return v.matches[0][0] == v.str
}

func (v variableReference) references() []string {
	var out []string
	for _, m := range v.matches {
		out = append(out, m[1])
	}
	return out
}

func (v variableReference) mapKey() string {
	return v.path.String()
}

func Interpolate(in dyn.Value) (out dyn.Value, err error) {
	return interpolation{in: in}.interpolate()
}

type interpolation struct {
	in dyn.Value

	refs     map[string]variableReference
	resolved map[string]dyn.Value
}

func (i interpolation) interpolate() (out dyn.Value, err error) {
	err = i.collectVariableReferences()
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Initialize map for resolved variables.
	// We use this for memoization.
	i.resolved = make(map[string]dyn.Value)

	// Resolve each variable reference (in order).
	keys := maps.Keys(i.refs)
	sort.Strings(keys)

	for _, key := range keys {
		_, err := i.resolve(key, []string{key})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	out, err = i.replaceVariableReferences()
	if err != nil {
		return dyn.InvalidValue, err
	}

	return out, nil
}

func (i *interpolation) collectVariableReferences() (err error) {
	i.refs = make(map[string]variableReference)

	// First walk the input to gather all values with a variable reference.
	_, err = dyn.Walk(i.in, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		s, ok := v.AsString()
		if !ok {
			// Skip non-string values.
			return v, nil
		}

		// Check if the string contains a variable reference.
		m := re.FindAllStringSubmatch(s, -1)
		if len(m) == 0 {
			// Skip strings without variable references.
			return v, nil
		}

		// Store the variable reference.
		ref := variableReference{
			value:   v,
			path:    p,
			str:     s,
			matches: m,
		}
		i.refs[ref.mapKey()] = ref
		return v, nil
	})

	return err
}

func (i *interpolation) resolve(key string, seen []string) (dyn.Value, error) {
	if v, ok := i.resolved[key]; ok {
		return v, nil
	}

	ref, ok := i.refs[key]
	if !ok {
		// Perform lookup in the input.
		// TODO hook into user specified function here
		return dyn.Get(i.in, key)
	}

	// This is an unresolved variable reference.
	deps := ref.references()

	// Resolve each of the dependencies, then interpolate them in the ref.
	resolved := make([]dyn.Value, len(deps))

	for j, dep := range deps {
		// Cycle detection.
		if slices.Contains(seen, dep) {
			return dyn.InvalidValue, fmt.Errorf(
				"cycle detected in field resolution: %s",
				strings.Join(append(seen, dep), " -> "),
			)
		}

		v, err := i.resolve(dep, append(seen, dep))
		if err != nil {
			return dyn.InvalidValue, err
		}

		resolved[j] = v
	}

	// Interpolate the resolved values.
	if ref.isPure() {
		// If the variable reference is pure, we can substitute it.
		// This is useful for interpolating values of non-string types.
		i.resolved[key] = resolved[0]
		return resolved[0], nil
	}

	// Not pure; perform string interpolation.
	for j := range ref.matches {
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
	i.resolved[key] = v
	return v, nil
}

func (i *interpolation) replaceVariableReferences() (dyn.Value, error) {
	// Walk the input and replace all variable references.
	return dyn.Walk(i.in, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		ref, ok := i.refs[p.String()]
		if !ok {
			// No variable reference; return the original value.
			return v, nil
		}

		// We have a variable reference; return the resolved value.
		return i.resolved[ref.mapKey()], nil
	})
}
