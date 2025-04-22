package dyn

import (
	"fmt"
	"slices"
)

// MapFunc is a function that maps a value to another value.
type MapFunc func(Path, Value) (Value, error)

// Foreach returns a [MapFunc] that applies the specified [MapFunc] to each
// value in a map or sequence and returns the new map or sequence.
// If the input is nil, it returns nil.
func Foreach(fn MapFunc) MapFunc {
	return func(p Path, v Value) (Value, error) {
		switch v.Kind() {
		case KindNil:
			return v, nil
		case KindMap:
			m := v.MustMap().Clone()
			for _, pair := range m.Pairs() {
				pk := pair.Key
				pv := pair.Value
				nv, err := fn(p.Append(Key(pk.MustString())), pv)
				if err != nil {
					return InvalidValue, err
				}
				m.SetLoc(pk.MustString(), pk.Locations(), nv)
			}
			return NewValue(m, v.Locations()), nil
		case KindSequence:
			s := slices.Clone(v.MustSequence())
			for i, value := range s {
				var err error
				s[i], err = fn(p.Append(Index(i)), value)
				if err != nil {
					return InvalidValue, err
				}
			}
			return NewValue(s, v.Locations()), nil
		default:
			return InvalidValue, fmt.Errorf("expected a map or sequence, found %s", v.Kind())
		}
	}
}

// Map applies a function to the value at the given path in the given value.
// It is identical to [MapByPath], except that it takes a string path instead of a [Path].
func Map(v Value, path string, fn MapFunc) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return MapByPath(v, p, fn)
}

// MapByPath applies a function to the value at the given path in the given value.
// It is identical to [MapByPattern], except that it takes a [Path] instead of a [Pattern].
// This means it only matches a single value, not a pattern of values.
func MapByPath(v Value, p Path, fn MapFunc) (Value, error) {
	return MapByPattern(v, NewPatternFromPath(p), fn)
}

// MapByPattern applies a function to the values whose paths match the given pattern in the given value.
// If successful, it returns the new value with all intermediate values copied and updated.
//
// If the pattern contains a key that doesn't exist, or an index that is out of bounds,
// it returns the original value and no error.
//
// If the pattern is invalid for the given value, it returns InvalidValue and an error.
func MapByPattern(v Value, p Pattern, fn MapFunc) (Value, error) {
	nv, err := visit(v, EmptyPath, p, visitOptions{
		fn: fn,
	})

	// Check for success.
	if err == nil {
		return nv, nil
	}

	// Return original value if:
	// - any map or sequence is a nil, or
	// - a key or index is missing
	if IsCannotTraverseNilError(err) || IsNoSuchKeyError(err) || IsIndexOutOfBoundsError(err) {
		return v, nil
	}

	return nv, err
}
