package dyn

// DropNulls returns v without the nil values of its maps, at any depth.
// Nil elements of sequences are kept.
func DropNulls(v Value) Value {
	switch v.Kind() {
	case KindMap:
		m := v.MustMap()
		out := newMappingWithSize(m.Len())
		for _, pair := range m.Pairs() {
			if pair.Value.Kind() == KindNil {
				continue
			}
			out.SetLoc(pair.Key.MustString(), pair.Key.Locations(), DropNulls(pair.Value))
		}
		v.v = out
	case KindSequence:
		s := v.MustSequence()
		out := make([]Value, len(s))
		for i := range s {
			out[i] = DropNulls(s[i])
		}
		v.v = out
	default:
		// Leaf node.
	}
	return v
}
