package config

// Walk walks the configuration tree and calls the given function on each node.
// This given function must return the new value for the node. Traversal is depth-first.
func Walk(v Value, fn func(Value) (Value, error)) (Value, error) {
	var err error

	switch v.Kind() {
	case KindMap:
		m := v.v.(map[string]Value)
		out := make(map[string]Value, len(m))
		for k, v := range m {
			out[k], err = Walk(v, fn)
			if err != nil {
				return NilValue, err
			}
		}
		v.v = out
	case KindSequence:
		s := v.v.([]Value)
		out := make([]Value, len(s))
		for i, v := range s {
			out[i], err = Walk(v, fn)
			if err != nil {
				return NilValue, err
			}
		}
		v.v = out
	}

	v, err = fn(v)
	if err != nil {
		return NilValue, err
	}

	return v, nil
}
