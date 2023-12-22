package dyn

import (
	"fmt"
	"maps"
)

func (v Value) SetByPath(p Path, value Value) (Value, error) {
	return v.set(EmptyPath, p, value)
}

func (v Value) Set(path string, value Value) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return v.set(EmptyPath, p, value)
}

func (v Value) set(prefix, suffix Path, nv Value) (Value, error) {
	if len(suffix) == 0 {
		return nv, nil
	}

	component := suffix[0]
	prefix = prefix.Append(component)
	suffix = suffix[1:]

	// Resolve first component.
	switch v.k {
	case KindMap:
		// Expect a key to be set if this is a map.
		if len(component.key) == 0 {
			return InvalidValue, fmt.Errorf("expected a key index at %s", prefix)
		}

		// Recurse on set to get a new map entry.
		m := v.MustMap()
		nv, err := m[component.key].set(prefix, suffix, nv)
		if err != nil {
			return InvalidValue, err
		}

		// Return an updated map value.
		m = maps.Clone(m)
		m[component.key] = nv
		return Value{
			v: m,
			k: KindMap,
			l: v.l,
		}, nil

	default:
		return InvalidValue, fmt.Errorf("expected a map under %s", prefix)
	}
}
