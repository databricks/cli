package dyn

import (
	"fmt"
	"slices"
)

// Set assigns a new value at the specified path in the specified value.
// It is identical to [SetByPath], except that it takes a string path instead of a [Path].
func Set(v Value, path string, nv Value) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return SetByPath(v, p, nv)
}

// SetByPath assigns the given value at the specified path in the specified value.
// If successful, it returns the new value with all intermediate values copied and updated.
// If the path doesn't exist, it returns InvalidValue and an error.
func SetByPath(v Value, p Path, nv Value) (Value, error) {
	lp := len(p)
	if lp == 0 {
		return nv, nil
	}

	component := p[lp-1]
	p = p[:lp-1]

	return visit(v, EmptyPath, NewPatternFromPath(p), visitOptions{
		fn: func(prefix Path, v Value) (Value, error) {
			path := append(prefix, component)

			switch {
			case component.isKey():
				// Expect a map to be set if this is a key.
				m, ok := v.AsMap()
				if !ok {
					return InvalidValue, fmt.Errorf("expected a map to index %q, found %s", path, v.Kind())
				}

				// Return an updated map value.
				m = m.Clone()
				m.SetLoc(component.key, nil, nv)
				return Value{
					v: m,
					k: KindMap,
					l: v.l,
				}, nil

			case component.isIndex():
				// Expect a sequence to be set if this is an index.
				s, ok := v.AsSequence()
				if !ok {
					return InvalidValue, fmt.Errorf("expected a sequence to index %q, found %s", path, v.Kind())
				}

				// Lookup current value in the sequence.
				if component.index < 0 || component.index >= len(s) {
					return InvalidValue, indexOutOfBoundsError{prefix}
				}

				// Return an updated sequence value.
				s = slices.Clone(s)
				s[component.index] = nv
				return Value{
					v: s,
					k: KindSequence,
					l: v.l,
				}, nil

			default:
				panic("invalid component")
			}
		},
	})
}
