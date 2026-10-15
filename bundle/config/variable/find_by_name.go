package variable

import (
	"fmt"
	"iter"
)

// findByName returns the single item whose name is name. typeName is only
// used in error messages; it keeps the wording the lookups have always used.
func findByName[T any](items iter.Seq2[*T, error], name, typeName string, nameOf func(*T) string) (*T, error) {
	var matches []*T
	for item, err := range items {
		if err != nil {
			return nil, err
		}
		if nameOf(item) == name {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("%s named '%s' does not exist", typeName, name)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("there are %d instances of %s named '%s'", len(matches), typeName, name)
	}
	return matches[0], nil
}

// sliceSeq adapts a slice of values to the iterator form findByName expects.
func sliceSeq[T any](items []T) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for i := range items {
			if !yield(&items[i], nil) {
				return
			}
		}
	}
}
