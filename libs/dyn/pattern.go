package dyn

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Pattern represents a matcher for paths in a [Value] configuration tree.
// It is used by [MapByPattern] to apply a function to the values whose paths match the pattern.
// Every [Path] is a valid [Pattern] that matches a single unique path.
// The reverse is not true; not every [Pattern] is a valid [Path], as patterns may contain wildcards.
type Pattern []patternComponent

func (p Pattern) String() string {
	buf := strings.Builder{}
	first := true

	for _, c := range p {
		switch c := c.(type) {
		case anyKeyComponent:
			if !first {
				buf.WriteString(".")
			}
			buf.WriteString("*")
		case anyIndexComponent:
			buf.WriteString("[*]")
		case pathComponent:
			if c.isKey() {
				if !first {
					buf.WriteString(".")
				}
				buf.WriteString(c.Key())
			} else {
				buf.WriteString(fmt.Sprintf("[%d]", c.Index()))
			}
		default:
			buf.WriteString("???")
		}

		first = false
	}
	return buf.String()
}

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

// Split <pattern>.<string_key> into <pattern> and <string_key>
// The last component must be dyn.Key() and there must be at least two components.
func (p Pattern) SplitKey() (Pattern, string) {
	if len(p) <= 1 {
		return nil, ""
	}
	parent := p[:len(p)-1]
	leaf := p[len(p)-1]
	pc, ok := leaf.(pathComponent)
	if !ok {
		return nil, ""
	}
	key := pc.Key()
	if key == "" {
		return nil, ""
	}
	return parent, key
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

type expectedMapError struct {
	p Path
	v Value
}

func (e expectedMapError) Error() string {
	return fmt.Sprintf("expected a map at %q, found %s", e.p, e.v.Kind())
}

func IsExpectedMapError(err error) bool {
	var target expectedMapError
	return errors.As(err, &target)
}

type expectedSequenceError struct {
	p Path
	v Value
}

func (e expectedSequenceError) Error() string {
	return fmt.Sprintf("expected a sequence at %q, found %s", e.p, e.v.Kind())
}

func IsExpectedSequenceError(err error) bool {
	var target expectedSequenceError
	return errors.As(err, &target)
}

// This function implements the patternComponent interface.
func (c anyKeyComponent) visit(v Value, prefix Path, suffix Pattern, opts visitOptions) (Value, error) {
	m, ok := v.AsMap()
	if !ok {
		return InvalidValue, expectedMapError{p: prefix, v: v}
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

		m.SetLoc(pk.MustString(), pk.Locations(), nv)
	}

	return NewValue(m, v.Locations()), nil
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
		return InvalidValue, expectedSequenceError{p: prefix, v: v}
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

	return NewValue(s, v.Locations()), nil
}
