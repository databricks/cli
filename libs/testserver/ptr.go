package testserver

// val returns the value p points to, or the zero value if p is nil. Request and
// state types are sdk-go JSON mirrors whose optional fields are pointers.
func val[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// nonZero returns a pointer to v, or nil if v is the zero value, so the field is
// omitted from JSON like an unset value field with omitempty.
func nonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
