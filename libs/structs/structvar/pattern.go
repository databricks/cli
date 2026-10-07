package structvar

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/structs/structpath"
)

// ForEach calls fn for every value in x that matches pattern, like dyn.MapByPattern
// visits them: map keys in order, wildcards skip values the rest of the pattern does
// not match, and a missing key or index outside a wildcard visits nothing.
//
// fn may change the configuration at or below the visited path (e.g. with [Set]),
// but must not add or remove the keys and elements being iterated.
func ForEach(x View, pattern *structpath.PatternNode, fn func(*structpath.PathNode, View) error) error {
	err := forEach(x, nil, pattern.AsSlice(), fn)
	if errors.Is(err, errNoMatch) {
		return nil
	}
	return err
}

// errNoMatch means the pattern did not match: a key or index is missing or a value is nil.
var errNoMatch = errors.New("no match")

// errUnexpectedKind means a wildcard was applied to a value of the wrong kind.
type errUnexpectedKind struct {
	msg string
}

func (e errUnexpectedKind) Error() string {
	return e.msg
}

func forEach(x View, prefix *structpath.PathNode, pattern []*structpath.PatternNode, fn func(*structpath.PathNode, View) error) error {
	if len(pattern) == 0 {
		return fn(prefix, x)
	}

	c := pattern[0]
	rest := pattern[1:]
	kind := x.Kind()

	if c.DotStar() {
		if kind != KindMap {
			return errUnexpectedKind{fmt.Sprintf("expected a map at %q, found %s", prefix, kind)}
		}
		for k, child := range x.MapItems() {
			if err := forEach(child, structpath.NewStringKey(prefix, k), rest, fn); err != nil && !isNoMatch(err) {
				return err
			}
		}
		return nil
	}

	if c.BracketStar() {
		if kind != KindSequence {
			return errUnexpectedKind{fmt.Sprintf("expected a sequence at %q, found %s", prefix, kind)}
		}
		for i, child := range x.Sequence() {
			if err := forEach(child, structpath.NewIndex(prefix, i), rest, fn); err != nil && !isNoMatch(err) {
				return err
			}
		}
		return nil
	}

	if key, ok := c.StringKey(); ok {
		path := structpath.NewStringKey(prefix, key)
		switch kind {
		case KindMap:
		case KindNil:
			return errNoMatch
		default:
			return fmt.Errorf("expected a map to index %q, found %s", path, kind)
		}
		child := x.Get(key)
		if !child.IsValid() {
			return errNoMatch
		}
		return forEach(child, path, rest, fn)
	}

	index, ok := c.Index()
	if !ok {
		return fmt.Errorf("unsupported pattern component %q", c)
	}
	path := structpath.NewIndex(prefix, index)
	switch kind {
	case KindSequence:
	case KindNil:
		return errNoMatch
	default:
		return fmt.Errorf("expected a sequence to index %q, found %s", path, kind)
	}
	child := x.Index(index)
	if !child.IsValid() {
		return errNoMatch
	}
	return forEach(child, path, rest, fn)
}

func isNoMatch(err error) bool {
	if errors.Is(err, errNoMatch) {
		return true
	}
	_, ok := errors.AsType[errUnexpectedKind](err)
	return ok
}
