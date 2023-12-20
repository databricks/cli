package config

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

type noSuchKeyError struct {
	p Path
}

func (e noSuchKeyError) Error() string {
	return fmt.Sprintf("no such key: %s", e.p)
}

func IsNoSuchKeyError(err error) bool {
	var target noSuchKeyError
	return errors.As(err, &target)
}

func (v Value) TransformByPath(p Path, value Value) (Value, error) {
	return v.set(EmptyPath, p, value)
}

func (v Value) Transform(path string, fn func(Value) (Value, error)) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return v.transform(EmptyPath, p, fn)
}

func (v Value) transform(prefix, suffix Path, fn func(Value) (Value, error)) (Value, error) {
	if len(suffix) == 0 {
		return fn(v)
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

		// Lookup current value in the map.
		m := v.MustMap()
		nv, ok := m[component.key]
		if !ok {
			return InvalidValue, noSuchKeyError{prefix}
		}

		// Recursively transform the value.
		nv, err := nv.transform(prefix, suffix, fn)
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

	case KindSequence:
		// Expect an index to be set if this is a sequence.
		if len(component.key) > 0 {
			return InvalidValue, fmt.Errorf("expected an index at %s", prefix)
		}

		// Lookup current value in the sequence.
		s := v.MustSequence()
		if component.index < 0 || component.index >= len(s) {
			return InvalidValue, fmt.Errorf("index out of bounds under %s", prefix)
		}

		// Recursively transform the value.
		nv, err := s[component.index].transform(prefix, suffix, fn)
		if err != nil {
			return InvalidValue, err
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
		return InvalidValue, fmt.Errorf("expected a map or sequence at %s", prefix)
	}
}
