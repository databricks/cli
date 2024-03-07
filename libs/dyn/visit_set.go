package dyn

// Set assigns a new value at the specified path in the specified value.
// It is identical to [SetByPath], except that it takes a string path instead of a [Path].
func Set(v Value, path string, nv Value) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return SetByPath(v, p, nv)
}

// SetByPath assigns the given value at the specified path in the specified value.
// If successful, it returns the new value with all intermediate values copied and updated.
// If the path doesn't exist, it returns InvalidValue and an error.
func SetByPath(v Value, p Path, nv Value) (Value, error) {
	return visit(v, EmptyPath, p, visitOptions{
		fn: func(_ Path, _ Value) (Value, error) {
			// Return the incoming value to set it.
			return nv, nil
		},
		allowMissingKeyInMap: true,
	})
}
