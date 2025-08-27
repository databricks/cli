package dyn

import (
	"maps"
	"slices"
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
	pairs []Pair
	index map[string]int
}

// NewMapping creates a new empty Mapping.
func NewMapping() Mapping {
	return Mapping{
		pairs: make([]Pair, 0),
		index: make(map[string]int),
	}
}

// NewMappingFromPairs computes a [Mapping] from a list of [Pair]s. The index
// map does not need to be provided since that will be computed from the
// key-value pairs provided.
func NewMappingFromPairs(pairs []Pair) Mapping {
	index := make(map[string]int)
	for i, p := range pairs {
		index[p.Key.MustString()] = i
	}

	return Mapping{
		pairs: pairs,
		index: index,
	}
}

// newMappingWithSize creates a new Mapping preallocated to the specified size.
func newMappingWithSize(size int) Mapping {
	return Mapping{
		pairs: make([]Pair, 0, size),
		index: make(map[string]int, size),
	}
}

// newMappingFromGoMap creates a new Mapping from a Go map of string keys and dynamic values.
func newMappingFromGoMap(vin map[string]Value) Mapping {
	m := newMappingWithSize(len(vin))
	for k, v := range vin {
		m.SetLoc(k, nil, v)
	}
	return m
}

// Pairs returns all the key-value pairs in the Mapping. The pairs are sorted by
// their key in lexicographic order.
func (m Mapping) Pairs() []Pair {
	return m.pairs
}

// Len returns the number of key-value pairs in the Mapping.
func (m Mapping) Len() int {
	return len(m.pairs)
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
	if i, ok := m.index[skey]; ok {
		return m.pairs[i], true
	}
	return Pair{}, false
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
// If the key already exists, the value is updated. The location loc is ignored.
// If the key does not exist, a new key-value pair is added.
func (m *Mapping) SetLoc(skey string, loc []Location, value Value) {
	// If the key already exists, update the value.
	if i, ok := m.index[skey]; ok {
		m.pairs[i].Value = value
		return
	}

	// Otherwise, add a new pair.
	m.pairs = append(m.pairs, Pair{NewValue(skey, loc), value})
	if m.index == nil {
		m.index = make(map[string]int)
	}
	m.index[skey] = len(m.pairs) - 1
}

// Keys returns all the keys in the Mapping.
func (m Mapping) Keys() []Value {
	keys := make([]Value, 0, len(m.pairs))
	for _, p := range m.pairs {
		keys = append(keys, p.Key)
	}
	return keys
}

// Values returns all the values in the Mapping.
func (m Mapping) Values() []Value {
	values := make([]Value, 0, len(m.pairs))
	for _, p := range m.pairs {
		values = append(values, p.Value)
	}
	return values
}

// Clone creates a shallow copy of the Mapping.
func (m Mapping) Clone() Mapping {
	return Mapping{
		pairs: slices.Clone(m.pairs),
		index: maps.Clone(m.index),
	}
}

// Merge merges the key-value pairs from another Mapping into the current Mapping.
func (m *Mapping) Merge(n Mapping) {
	for _, p := range n.pairs {
		m.SetLoc(p.Key.MustString(), p.Key.Locations(), p.Value)
	}
}
