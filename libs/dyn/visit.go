package dyn

import (
	"errors"
	"fmt"
	"slices"
)

type noSuchKeyError struct {
	p Path
}

func (e noSuchKeyError) Error() string {
	return fmt.Sprintf("key not found at %q", e.p)
}

func IsNoSuchKeyError(err error) bool {
	var target noSuchKeyError
	return errors.As(err, &target)
}

type indexOutOfBoundsError struct {
	p Path
}

func (e indexOutOfBoundsError) Error() string {
	return fmt.Sprintf("index out of bounds at %q", e.p)
}

func IsIndexOutOfBoundsError(err error) bool {
	var target indexOutOfBoundsError
	return errors.As(err, &target)
}

type visitOptions struct {
	// The function to apply to the value once found.
	//
	// If this function returns the same value as it receives as argument,
	// the original visit function call returns the original value unmodified.
	//
	// If this function returns a new value, the original visit function call
	// returns a value with all the intermediate values updated.
	//
	// If this function returns an error, the original visit function call
	// returns this error and the value is left unmodified.
	fn func(Path, Value) (Value, error)
}

func visit(v Value, prefix Path, suffix Pattern, opts visitOptions) (Value, error) {
	if len(suffix) == 0 {
		return opts.fn(prefix, v)
	}

	// Initialize prefix if it is empty.
	// It is pre-allocated to its maximum size to avoid additional allocations.
	if len(prefix) == 0 {
		prefix = make(Path, 0, len(suffix))
	}

	component := suffix[0]
	suffix = suffix[1:]

	// Visit the value with the current component.
	return component.visit(v, prefix, suffix, opts)
}

func (component pathComponent) visit(v Value, prefix Path, suffix Pattern, opts visitOptions) (Value, error) {
	path := append(prefix, component)

	switch {
	case component.isKey():
		// Expect a map to be set if this is a key.
		m, ok := v.AsMap()
		if !ok {
			return InvalidValue, fmt.Errorf("expected a map to index %q, found %s", path, v.Kind())
		}

		// Lookup current value in the map.
		ev, ok := m.GetByString(component.key)
		if !ok {
			return InvalidValue, noSuchKeyError{path}
		}

		// Recursively transform the value.
		nv, err := visit(ev, path, suffix, opts)
		if err != nil {
			return InvalidValue, err
		}

		// Return the original value if the value hasn't changed.
		if nv.eq(ev) {
			return v, nil
		}

		// Return an updated map value.
		m = m.Clone()
		m.Set(V(component.key), nv)
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
			return InvalidValue, indexOutOfBoundsError{path}
		}

		// Recursively transform the value.
		ev := s[component.index]
		nv, err := visit(ev, path, suffix, opts)
		if err != nil {
			return InvalidValue, err
		}

		// Return the original value if the value hasn't changed.
		if nv.eq(ev) {
			return v, nil
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
}
