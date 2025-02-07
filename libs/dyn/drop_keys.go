package dyn

func DropKeys(v Value, drop []string) (Value, error) {
	var err error
	nv, err := Walk(v, func(p Path, v Value) (Value, error) {
		if len(p) == 0 {
			return v, nil
		}

		// Check if this key should be dropped.
		for _, key := range drop {
			if p[0].Key() != key {
				continue
			}

			return InvalidValue, ErrDrop
		}

		// Pass through all other values.
		return v, ErrSkip
	})
	if err != nil {
		return InvalidValue, err
	}

	return nv, nil
}
