package dyn

import (
	"fmt"
	"slices"
)

// Pattern represents a matcher for paths in a [Value] configuration tree.
// It is used by [MapByPattern] to apply a function to the values whose paths match the pattern.
// Every [Path] is a valid [Pattern] that matches a single unique path.
// The reverse is not true; not every [Pattern] is a valid [Path], as patterns may contain wildcards.
type Pattern []patternComponent

// A pattern component can visit a [Value] and recursively call into [visit] for matching elements.
// Fixed components can match a single key or index, while wildcards can match any key or index.
type patternComponent interface {
	visit(v Value, prefix Path, suffix Pattern, opts visitOptions) (Value, error)
}

// NewPattern returns a new pattern from the given components.
// The individual components may be created with [Key], [Index], or [Any].
func NewPattern(cs ...patternComponent) Pattern {
	return cs
}

// NewPatternFromPath returns a new pattern from the given path.
func NewPatternFromPath(p Path) Pattern {
	cs := make(Pattern, len(p))
	for i, c := range p {
		cs[i] = c
	}
	return cs
}

// Append appends the given components to the pattern.
func (p Pattern) Append(cs ...patternComponent) Pattern {
	out := make(Pattern, len(p)+len(cs))
	copy(out, p)
	copy(out[len(p):], cs)
	return out
}

type anyKeyComponent struct{}

// AnyKey returns a pattern component that matches any key.
func AnyKey() patternComponent {
	return anyKeyComponent{}
}

// This function implements the patternComponent interface.
func (c anyKeyComponent) visit(v Value, prefix Path, suffix Pattern, opts visitOptions) (Value, error) {
	m, ok := v.AsMapping()
	if !ok {
		return InvalidValue, fmt.Errorf("expected a map at %q, found %s", prefix, v.Kind())
	}

	m = m.Clone()
	for _, pair := range m.Pairs() {
		pk := pair.Key
		pv := pair.Value

		var err error
		nv, err := visit(pv, append(prefix, Key(pk.MustString())), suffix, opts)
		if err != nil {
			// Leave the value intact if the suffix pattern didn't match any value.
			if IsNoSuchKeyError(err) || IsIndexOutOfBoundsError(err) {
				continue
			}
			return InvalidValue, err
		}

		m.Set(pk, nv)
	}

	return NewValue(m, v.Location()), nil
}

type anyIndexComponent struct{}

// AnyIndex returns a pattern component that matches any index.
func AnyIndex() patternComponent {
	return anyIndexComponent{}
}

// This function implements the patternComponent interface.
func (c anyIndexComponent) visit(v Value, prefix Path, suffix Pattern, opts visitOptions) (Value, error) {
	s, ok := v.AsSequence()
	if !ok {
		return InvalidValue, fmt.Errorf("expected a sequence at %q, found %s", prefix, v.Kind())
	}

	s = slices.Clone(s)
	for i, value := range s {
		var err error
		nv, err := visit(value, append(prefix, Index(i)), suffix, opts)
		if err != nil {
			// Leave the value intact if the suffix pattern didn't match any value.
			if IsNoSuchKeyError(err) || IsIndexOutOfBoundsError(err) {
				continue
			}
			return InvalidValue, err
		}
		s[i] = nv
	}

	return NewValue(s, v.Location()), nil
}
