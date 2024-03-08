package dyn

// Get returns the value inside the specified value at the specified path.
// It is identical to [GetByPath], except that it takes a string path instead of a [Path].
func Get(v Value, path string) (Value, error) {
	p, err := NewPathFromString(path)
	if err != nil {
		return InvalidValue, err
	}
	return GetByPath(v, p)
}

// GetByPath returns the value inside the specified value at the specified path.
// If the path doesn't exist, it returns InvalidValue and an error.
func GetByPath(v Value, p Path) (Value, error) {
	out := InvalidValue
	_, err := visit(v, EmptyPath, NewPatternFromPath(p), visitOptions{
		fn: func(_ Path, ev Value) (Value, error) {
			// Capture the value argument to return it.
			out = ev
			return ev, nil
		},
	})
	return out, err
}
