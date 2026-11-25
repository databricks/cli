package config

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"sync"
)

// ResourceTypeRegistry holds at most one value per resource type. A package keeps one in
// a package-level variable, and the file of each type that the package handles registers
// its value from init().
type ResourceTypeRegistry[T any] struct {
	name          string
	registrations []resourceTypeRegistration[T]
	ordered       func() []resourceTypeRegistration[T]
}

type resourceTypeRegistration[T any] struct {
	key   string
	value T
}

// NewResourceTypeRegistry returns an empty registry. name describes what is registered,
// for example "pre-deploy check", and is used in panic messages.
func NewResourceTypeRegistry[T any](name string) *ResourceTypeRegistry[T] {
	r := &ResourceTypeRegistry[T]{name: name}
	r.ordered = sync.OnceValue(func() []resourceTypeRegistration[T] {
		return r.order(allResourceTypes())
	})
	return r
}

// Register records value for the resource type under key. Call it from init() in the
// file of that type. It only records the value; it is checked on first use.
func (r *ResourceTypeRegistry[T]) Register(key string, value T) {
	r.registrations = append(r.registrations, resourceTypeRegistration[T]{key: key, value: value})
}

// All yields the key and value of every registration, in the canonical order of resource
// types, whatever the order in which they were registered.
//
// On first use, it panics if a registration is for a key that is not a resource type or
// if a key is registered twice, because each of these is a mistake in the code that
// registers.
func (r *ResourceTypeRegistry[T]) All() iter.Seq2[string, T] {
	return func(yield func(string, T) bool) {
		for _, reg := range r.ordered() {
			if !yield(reg.key, reg.value) {
				return
			}
		}
	}
}

// Get returns the value registered for key, and false if there is none. It panics like
// [ResourceTypeRegistry.All].
func (r *ResourceTypeRegistry[T]) Get(key string) (T, bool) {
	for k, v := range r.All() {
		if k == key {
			return v, true
		}
	}
	var zero T
	return zero, false
}

func (r *ResourceTypeRegistry[T]) order(types []ResourceType) []resourceTypeRegistration[T] {
	typeOrder := map[string]int{}
	for i, rt := range types {
		typeOrder[rt.Key] = i
	}

	regs := slices.Clone(r.registrations)
	seen := map[string]bool{}
	for _, reg := range regs {
		if _, ok := typeOrder[reg.key]; !ok {
			panic(fmt.Sprintf("%s registered for unknown resource type %q", r.name, reg.key))
		}
		if seen[reg.key] {
			panic(fmt.Sprintf("%s registered more than once for resource type %q", r.name, reg.key))
		}
		seen[reg.key] = true
	}
	slices.SortFunc(regs, func(a, b resourceTypeRegistration[T]) int {
		return cmp.Compare(typeOrder[a.key], typeOrder[b.key])
	})
	return regs
}
