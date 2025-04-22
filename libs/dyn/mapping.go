package dyn

import (
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/libs/utils"
)

// Pair represents a single key-value pair in a Mapping.
type Pair struct {
	Key   Value
	Value Value
}

// Mapping represents a key-value map of dynamic values.
// It exists because plain Go maps cannot use dynamic values for keys.
// We need to use dynamic values for keys because it lets us associate metadata
// with keys (i.e. their definition location). Keys must be strings.
type Mapping struct {
	data         map[string]Value
	keyLocations map[string][]Location
}

// NewMapping creates a new empty Mapping.
func NewMapping() Mapping {
	return Mapping{}
}

// NewMappingWithSize creates a new Mapping preallocated to the specified size.
func NewMappingWithSize(size int) Mapping {
	return newMappingWithSize(size)
}

// newMappingWithSize creates a new Mapping preallocated to the specified size.
func newMappingWithSize(size int) Mapping {
	return Mapping{
		data: make(map[string]Value, size),
		// Do not initialize keyLocations, because in many contexts it's not going to be filled
		// e.g. when constructing Mapping from map[string]Value.
	}
}

// NewMappingFromGoMap creates a new Mapping from a Go map of string keys and dynamic values.
func NewMappingFromGoMap(vin map[string]Value) Mapping {
	return newMappingFromGoMap(vin)
}

// newMappingFromGoMap creates a new Mapping from a Go map of string keys and dynamic values.
func newMappingFromGoMap(vin map[string]Value) Mapping {
	return Mapping{
		data: maps.Clone(vin),
	}
}

// Pairs returns all the key-value pairs in the Mapping. The pairs are sorted by
// their key in lexicographic order.
func (m Mapping) Pairs() []Pair {
	pairs := make([]Pair, 0, len(m.data))
	for _, k := range utils.SortedKeys(m.data) {
		pairs = append(pairs, Pair{
			Key:   NewValue(k, m.keyLocations[k]),
			Value: m.data[k],
		})
	}
	return pairs
}

// Len returns the number of key-value pairs in the Mapping.
func (m Mapping) Len() int {
	return len(m.data)
}

// GetPair returns the key-value pair with the specified key.
// It also returns a boolean indicating whether the pair was found.
func (m Mapping) GetPair(key Value) (Pair, bool) {
	skey, ok := key.AsString()
	if !ok {
		return Pair{}, false
	}
	return m.GetPairByString(skey)
}

// GetPairByString returns the key-value pair with the specified string key.
// It also returns a boolean indicating whether the pair was found.
func (m Mapping) GetPairByString(skey string) (Pair, bool) {
	val, ok := m.data[skey]
	if !ok {
		return Pair{}, false
	}

	return Pair{
		Key:   NewValue(skey, m.keyLocations[skey]),
		Value: val,
	}, true
}

// Get returns the value associated with the specified key.
// It also returns a boolean indicating whether the value was found.
func (m Mapping) Get(key Value) (Value, bool) {
	p, ok := m.GetPair(key)
	return p.Value, ok
}

// GetByString returns the value associated with the specified string key.
// It also returns a boolean indicating whether the value was found.
func (m *Mapping) GetByString(skey string) (Value, bool) {
	p, ok := m.GetPairByString(skey)
	return p.Value, ok
}

// Set sets the value for the given key in the mapping.
// If the key already exists, the value is updated.
// If the key does not exist, a new key-value pair is added.
// The key must be a string, otherwise an error is returned.
func (m *Mapping) Set(key, value Value) error {
	skey, ok := key.AsString()
	if !ok {
		return fmt.Errorf("key must be a string, got %s", key.Kind())
	}

	m.set(skey, key.l, value)
	return nil
}

func (m *Mapping) set(key string, keyloc []Location, value Value) {
	if m.data == nil {
		m.data = make(map[string]Value)
	}
	m.data[key] = value

	if len(keyloc) == 0 {
		delete(m.keyLocations, key)
	} else {
		if m.keyLocations == nil {
			m.keyLocations = make(map[string][]Location)
		}
		m.keyLocations[key] = slices.Clone(keyloc)
	}
}

// Keys returns all the keys in the Mapping.
func (m Mapping) Keys() []Value {
	keys := make([]Value, 0, len(m.data))
	for _, k := range utils.SortedKeys(m.data) {
		keys = append(keys, NewValue(k, m.keyLocations[k]))
	}
	return keys
}

// Values returns all the values in the Mapping.
func (m Mapping) Values() []Value {
	values := make([]Value, 0, len(m.data))
	for _, k := range utils.SortedKeys(m.data) {
		values = append(values, m.data[k])
	}
	return values
}

// Clone creates a shallow copy of the Mapping.
func (m Mapping) Clone() Mapping {
	return Mapping{
		data:         maps.Clone(m.data),
		keyLocations: maps.Clone(m.keyLocations),
	}
}

// Merge merges the key-value pairs from another Mapping into the current Mapping.
func (m *Mapping) Merge(n Mapping) {
	if len(m.data) == 0 {
		m.data = maps.Clone(n.data)
		m.keyLocations = maps.Clone(n.keyLocations)
		return
	}
	for key, value := range n.data {
		m.set(key, n.keyLocations[key], value)
	}
}
