package dyn

import (
	"fmt"
	"maps"
	"slices"
)

// MapFunc is a function that maps a value to another value.
type MapFunc func(Value) (Value, error)

// Foreach returns a [MapFunc] that applies the specified [MapFunc] to each
// value in a map or sequence and returns the new map or sequence.
func Foreach(fn MapFunc) MapFunc {
	return func(v Value) (Value, error) {
		switch v.Kind() {
		case KindMap:
			m := maps.Clone(v.MustMap())
			for key, value := range m {
				var err error
				m[key], err = fn(value)
				if err != nil {
					return InvalidValue, err
				}
			}
			return NewValue(m, v.Location()), nil
		case KindSequence:
			s := slices.Clone(v.MustSequence())
			for i, value := range s {
				var err error
				s[i], err = fn(value)
				if err != nil {
					return InvalidValue, err
				}
			}
			return NewValue(s, v.Location()), nil
		default:
			return InvalidValue, fmt.Errorf("expected a map or sequence, found %s", v.Kind())
		}
	}
}

// Map applies the given function to the value at the specified path in the specified value.
// It is identical to [MapByPath], except that it takes a string path instead of a [Path].
func Map(v Value, path string, fn MapFunc) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return MapByPath(v, p, fn)
}

// Map applies the given function to the value at the specified path in the specified value.
// If successful, it returns the new value with all intermediate values copied and updated.
//
// If the path contains a key that doesn't exist, or an index that is out of bounds,
// it returns the original value and no error. This is because setting a value at a path
// that doesn't exist is a no-op.
//
// If the path is invalid for the given value, it returns InvalidValue and an error.
func MapByPath(v Value, p Path, fn MapFunc) (Value, error) {
	nv, err := visit(v, EmptyPath, p, visitOptions{
		fn: fn,
	})

	// Check for success.
	if err == nil {
		return nv, nil
	}

	// Return original value if a key or index is missing.
	if IsNoSuchKeyError(err) || IsIndexOutOfBoundsError(err) {
		return v, nil
	}

	return nv, err
}
