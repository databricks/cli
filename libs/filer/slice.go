package filer

import "slices"

// sliceWithout returns a copy of the specified slice without element e, if it is present.
func sliceWithout[S []E, E comparable](s S, e E) S {
	s_ := slices.Clone(s)
	i := slices.Index(s_, e)
	if i >= 0 {
		s_ = slices.Delete(s_, i, i+1)
	}
	return s_
}
