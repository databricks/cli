package dyn

import (
	"fmt"
	"maps"
	"slices"
)

type Pair struct {
	Key   Value
	Value Value
}

type Mapping struct {
	pairs []Pair
	index map[string]int
}

func NewMapping() Mapping {
	return Mapping{
		pairs: make([]Pair, 0),
		index: make(map[string]int),
	}
}

func newMappingWithSize(size int) Mapping {
	return Mapping{
		pairs: make([]Pair, 0, size),
		index: make(map[string]int, size),
	}
}

func newMappingFromGoMap(vin map[string]Value) Mapping {
	m := newMappingWithSize(len(vin))
	for k, v := range vin {
		m.Set(V(k), v)
	}
	return m
}

func (m Mapping) Pairs() []Pair {
	return m.pairs
}

func (m Mapping) Keys() []Value {
	keys := make([]Value, 0, len(m.pairs))
	for _, p := range m.pairs {
		keys = append(keys, p.Key)
	}
	return keys
}

func (m Mapping) Values() []Value {
	values := make([]Value, 0, len(m.pairs))
	for _, p := range m.pairs {
		values = append(values, p.Value)
	}
	return values
}

func (m Mapping) Len() int {
	return len(m.pairs)
}

func (m Mapping) GetPairByString(skey string) (Pair, bool) {
	if i, ok := m.index[skey]; ok {
		return m.pairs[i], true
	}
	return Pair{}, false
}

func (m Mapping) GetPair(key Value) (Pair, bool) {
	skey, ok := key.AsString()
	if !ok {
		return Pair{}, false
	}
	return m.GetPairByString(skey)
}

func (m Mapping) Get(key Value) (Value, bool) {
	p, ok := m.GetPair(key)
	return p.Value, ok
}

func (m *Mapping) GetByString(skey string) (Value, bool) {
	p, ok := m.GetPairByString(skey)
	return p.Value, ok
}

func (m *Mapping) Set(key Value, value Value) error {
	skey, ok := key.AsString()
	if !ok {
		return fmt.Errorf("key must be a string, got %s", key.Kind())
	}

	// If the key already exists, update the value.
	if i, ok := m.index[skey]; ok {
		m.pairs[i].Value = value
		return nil
	}

	// Otherwise, add a new pair.
	m.pairs = append(m.pairs, Pair{key, value})
	if m.index == nil {
		m.index = make(map[string]int)
	}
	m.index[skey] = len(m.pairs) - 1
	return nil
}

func (m Mapping) Clone() Mapping {
	return Mapping{
		pairs: slices.Clone(m.pairs),
		index: maps.Clone(m.index),
	}
}

func (m *Mapping) Merge(n Mapping) {
	for _, p := range n.pairs {
		m.Set(p.Key, p.Value)
	}
}
