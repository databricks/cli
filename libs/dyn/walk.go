package dyn

import "errors"

// WalkValueFunc is the type of the function called by Walk to traverse the configuration tree.
type WalkValueFunc func(p Path, v Value) (Value, error)

// ErrDrop may be returned by WalkValueFunc to remove a value from the subtree.
var ErrDrop = errors.New("drop value from subtree")

// ErrSkip may be returned by WalkValueFunc to skip traversal of a subtree.
var ErrSkip = errors.New("skip traversal of subtree")

// Walk walks the configuration tree and calls the given function on each node.
// The callback may return ErrDrop to remove a value from the subtree.
// The callback may return ErrSkip to skip traversal of a subtree.
// If the callback returns another error, the walk is aborted, and the error is returned.
func Walk(v Value, fn func(p Path, v Value) (Value, error)) (Value, error) {
	return walk(v, EmptyPath, fn)
}

// Unexported counterpart to Walk.
// It carries the path leading up to the current node,
// such that it can be passed to the WalkValueFunc.
func walk(v Value, p Path, fn func(p Path, v Value) (Value, error)) (Value, error) {
	v, err := fn(p, v)
	if err != nil {
		if err == ErrSkip {
			return v, nil
		}
		return InvalidValue, err
	}

	switch v.Kind() {
	case KindMap:
		m := v.MustMap()
		out := newMappingWithSize(m.Len())
		for _, pair := range m.Pairs() {
			pk := pair.Key
			pv := pair.Value
			nv, err := walk(pv, append(p, Key(pk.MustString())), fn)
			if err == ErrDrop {
				continue
			}
			if err != nil {
				return InvalidValue, err
			}
			out.SetLoc(pk.MustString(), pk.Locations(), nv)
		}
		v.v = out
	case KindSequence:
		s := v.MustSequence()
		out := make([]Value, 0, len(s))
		for i := range s {
			nv, err := walk(s[i], append(p, Index(i)), fn)
			if err == ErrDrop {
				continue
			}
			if err != nil {
				return InvalidValue, err
			}
			out = append(out, nv)
		}
		v.v = out
	}

	return v, nil
}

// CollectLeafPaths traverses the value and returns all paths (as dot notation strings) to leaf nodes (non-map, non-sequence).
// The return value is not ordered.
func CollectLeafPaths(v Value) []string {
	var paths []string

	Walk(v, func(p Path, v Value) (Value, error) { //nolint:errcheck
		if len(p) == 0 {
			return v, nil
		}

		switch v.Kind() {
		case KindMap, KindSequence:
			// Ignore internal nodes.
		default:
			paths = append(paths, p.String())
		}

		return v, nil
	})

	return paths
}
