package dyn

// WalkReadOnly walks the configuration tree in readonly mode and calls the given function on each node.
// The callback may return ErrSkip to skip traversal of a subtree.
// If the callback returns another error, the walk is aborted, and the error is returned.
func WalkReadOnly(v Value, fn func(p Path, v Value) error) error {
	return walkReadOnly(v, EmptyPath, fn)
}

// Unexported counterpart to WalkReadOnly.
// It carries the path leading up to the current node,
// such that it can be passed to the callback function.
func walkReadOnly(v Value, p Path, fn func(p Path, v Value) error) error {
	if err := fn(p, v); err != nil {
		if err == ErrSkip {
			return nil
		}
		return err
	}

	switch v.Kind() {
	case KindMap:
		m := v.MustMap()
		for _, pair := range m.Pairs() {
			pk := pair.Key
			pv := pair.Value
			if err := walkReadOnly(pv, append(p, Key(pk.MustString())), fn); err != nil {
				return err
			}
		}
	case KindSequence:
		s := v.MustSequence()
		for i := range s {
			if err := walkReadOnly(s[i], append(p, Index(i)), fn); err != nil {
				return err
			}
		}
	}

	return nil
}
