package errs

import "errors"

type aggregateError struct {
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
	aggregateErr := &aggregateError{
		errors: make([]error, 0, n),
	}
	for _, err := range errors {
		if err != nil {
			aggregateErr.errors = append(aggregateErr.errors, err)
		}
	}
	return aggregateErr
}

func (ce *aggregateError) Error() string {
	var b []byte
	for i, err := range ce.errors {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (ce *aggregateError) Unwrap() error {
	return errorChain(ce.errors)
}

// Represents chained list of errors.
// Implements Error interface so that chain of errors
// can correctly work with errors.Is/As method
type errorChain []error

func (ec errorChain) Error() string {
	return ec[0].Error()
}

func (ec errorChain) Unwrap() error {
	if len(ec) == 1 {
		return nil
	}

	return ec[1:]
}

func (ec errorChain) As(target any) bool {
	return errors.As(ec[0], target)
}

func (ec errorChain) Is(target error) bool {
	return errors.Is(ec[0], target)
}
