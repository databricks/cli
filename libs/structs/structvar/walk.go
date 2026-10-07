package structvar

import (
	"errors"

	"github.com/databricks/cli/libs/structs/structpath"
)

// Walk calls fn for x and every value below it, parents before children, like
// [dyn.WalkReadOnly] visits a tree. Maps are visited in [View.MapItems] order.
// fn may return [ErrSkip] to skip the subtree of the visited value.
//
// fn may change the configuration at or below the visited path (e.g. with [Set]),
// but must not add or remove the keys and elements being iterated.
func Walk(x View, fn func(*structpath.PathNode, View) error) error {
	return walk(x, nil, fn)
}

func walk(x View, p *structpath.PathNode, fn func(*structpath.PathNode, View) error) error {
	if err := fn(p, x); err != nil {
		if errors.Is(err, ErrSkip) {
			return nil
		}
		return err
	}

	switch x.Kind() {
	case KindMap:
		for k, c := range x.MapItems() {
			if err := walk(c, structpath.NewStringKey(p, k), fn); err != nil {
				return err
			}
		}
	case KindSequence:
		for i, c := range x.Sequence() {
			if err := walk(c, structpath.NewIndex(p, i), fn); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

// ErrSkip can be returned by the function passed to [Walk] to skip the subtree of the
// visited value.
var ErrSkip = errors.New("skip traversal of subtree")
