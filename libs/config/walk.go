package config

import "errors"

// WalkValueFunc is the type of the function called by Walk to traverse the configuration tree.
type WalkValueFunc func(p Path, v Value) (Value, error)

// ErrDrop may be returned by WalkValueFunc to remove a value from the subtree.
var ErrDrop = errors.New("drop value from subtree")

// ErrSkip may be returned by WalkValueFunc to skip traversal of a subtree.
var ErrSkip = errors.New("skip traversal of subtree")

// Walk walks the configuration tree and calls the given function on each node.
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
		return NilValue, err
	}

	switch v.Kind() {
	case KindMap:
		m := v.v.(map[string]Value)
		out := make(map[string]Value, len(m))
		for k := range m {
			nv, err := walk(m[k], p.Append(Key(k)), fn)
			if err == ErrDrop {
				continue
			}
			if err != nil {
				return NilValue, err
			}
			out[k] = nv
		}
		v.v = out
	case KindSequence:
		s := v.v.([]Value)
		out := make([]Value, 0, len(s))
		for i := range s {
			nv, err := walk(s[i], p.Append(Index(i)), fn)
			if err == ErrDrop {
				continue
			}
			if err != nil {
				return NilValue, err
			}
			out = append(out, nv)
		}
		v.v = out
	}

	return v, nil
}
