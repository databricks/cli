package utils

// NonZero returns a pointer to v, or nil if v is the zero value of T. It
// mirrors `omitempty` on a value field: zero values are not sent.
func NonZero[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
