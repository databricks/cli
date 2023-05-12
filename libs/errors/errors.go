package errors

type combinedError struct {
	errors []error
}

func FromMany(errors ...error) error {
	n := 0
	for _, err := range errors {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	combinedErr := &combinedError{
		errors: make([]error, 0, n),
	}
	for _, err := range errors {
		if err != nil {
			combinedErr.errors = append(combinedErr.errors, err)
		}
	}
	return combinedErr
}

func (ce *combinedError) Error() string {
	var b []byte
	for i, err := range ce.errors {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (ce *combinedError) Unwrap() []error {
	return ce.errors
}
