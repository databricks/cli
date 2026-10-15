package listing

import "iter"

// Collect drains an sdk-go iterator into a slice of values, stopping at the
// first error. It replaces the previous SDK's ListAll helpers.
func Collect[T any](seq iter.Seq2[*T, error]) ([]T, error) {
	var result []T
	for v, err := range seq {
		if err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	return result, nil
}
