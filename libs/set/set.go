package set

import (
	"fmt"

	"golang.org/x/exp/maps"
)

type hashFunc[T any] func(a T) string

// Set struct represents set data structure
type Set[T any] struct {
	key  hashFunc[T]
	data map[string]T
}

// Values returns a slice of the set's values
func (s *Set[T]) Values() []T {
	return maps.Values(s.data)
}

// NewSetFromF initialise a new set with initial values and a hash function
// to define uniqueness of value
func NewSetFromF[T any](values []T, f hashFunc[T]) *Set[T] {
	s := &Set[T]{
		key:  f,
		data: make(map[string]T),
	}

	for _, v := range values {
		s.Add(v)
	}

	return s
}

// NewSetF initialise a new empty and a hash function
// to define uniqueness of value
func NewSetF[T any](f hashFunc[T]) *Set[T] {
	return NewSetFromF([]T{}, f)
}

// NewSetFrom initialise a new set with initial values which are comparable
func NewSetFrom[T comparable](values []T) *Set[T] {
	return NewSetFromF(values, func(item T) string {
		return fmt.Sprintf("%v", item)
	})
}

// NewSetFrom initialise a new empty set for comparable values
func NewSet[T comparable]() *Set[T] {
	return NewSetFrom([]T{})
}

func (s *Set[T]) addOne(item T) {
	s.data[s.key(item)] = item
}

// Add one or multiple items to set
func (s *Set[T]) Add(items ...T) {
	for _, i := range items {
		s.addOne(i)
	}
}

// Remove an item from set. No-op if the item does not exist
func (s *Set[T]) Remove(item T) {
	delete(s.data, s.key(item))
}

// Indicates if the item exists in the set
func (s *Set[T]) Has(item T) bool {
	_, ok := s.data[s.key(item)]
	return ok
}

// Size returns the number of elements in the set
func (s *Set[T]) Size() int {
	return len(s.data)
}

// Returns an iterable slice of values from set
func (s *Set[T]) Iter() []T {
	return maps.Values(s.data)
}
