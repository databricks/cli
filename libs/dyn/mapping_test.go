package dyn_test

import (
	"strconv"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestNewMapping(t *testing.T) {
	m := dyn.NewMapping()
	assert.Equal(t, 0, m.Len())
}

func TestMappingZeroValue(t *testing.T) {
	var m dyn.Mapping
	assert.Equal(t, 0, m.Len())

	value, ok := m.Get(dyn.V("key"))
	assert.Equal(t, dyn.InvalidValue, value)
	assert.False(t, ok)
	assert.Len(t, m.Keys(), 0)
	assert.Len(t, m.Values(), 0)
}

func TestMappingGet(t *testing.T) {
	var m dyn.Mapping
	m.SetLoc("key", nil, dyn.V("value"))
	assert.Equal(t, 1, m.Len())

	// Call GetPair
	p, ok := m.GetPair(dyn.V("key"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("key"), p.Key)
	assert.Equal(t, dyn.V("value"), p.Value)

	// Modify the value to make sure we're not getting a reference
	p.Value = dyn.V("newvalue")

	// Call GetPair with invalid key
	p, ok = m.GetPair(dyn.V(1234))
	assert.False(t, ok)
	assert.Equal(t, dyn.InvalidValue, p.Key)
	assert.Equal(t, dyn.InvalidValue, p.Value)

	// Call GetPair with non-existent key
	p, ok = m.GetPair(dyn.V("enoexist"))
	assert.False(t, ok)
	assert.Equal(t, dyn.InvalidValue, p.Key)
	assert.Equal(t, dyn.InvalidValue, p.Value)

	// Call GetPairByString
	p, ok = m.GetPairByString("key")
	assert.True(t, ok)
	assert.Equal(t, dyn.V("key"), p.Key)
	assert.Equal(t, dyn.V("value"), p.Value)

	// Modify the value to make sure we're not getting a reference
	p.Value = dyn.V("newvalue")

	// Call GetPairByString with with non-existent key
	p, ok = m.GetPairByString("enoexist")
	assert.False(t, ok)
	assert.Equal(t, dyn.InvalidValue, p.Key)
	assert.Equal(t, dyn.InvalidValue, p.Value)

	// Call Get
	value, ok := m.Get(dyn.V("key"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("value"), value)

	// Call Get with invalid key
	value, ok = m.Get(dyn.V(1234))
	assert.False(t, ok)
	assert.Equal(t, dyn.InvalidValue, value)

	// Call Get with non-existent key
	value, ok = m.Get(dyn.V("enoexist"))
	assert.False(t, ok)
	assert.Equal(t, dyn.InvalidValue, value)

	// Call GetByString
	value, ok = m.GetByString("key")
	assert.True(t, ok)
	assert.Equal(t, dyn.V("value"), value)

	// Call GetByString with non-existent key
	value, ok = m.GetByString("enoexist")
	assert.False(t, ok)
	assert.Equal(t, dyn.InvalidValue, value)
}

func TestMappingSetLoc(t *testing.T) {
	var m dyn.Mapping

	// Set a value
	m.SetLoc("key1", nil, dyn.V("foo"))
	assert.Equal(t, 1, m.Len())

	// Confirm the value
	value, ok := m.GetByString("key1")
	assert.True(t, ok)
	assert.Equal(t, dyn.V("foo"), value)

	// Set another value
	m.SetLoc("key2", nil, dyn.V("bar"))
	assert.Equal(t, 2, m.Len())

	// Confirm the value
	value, ok = m.Get(dyn.V("key2"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("bar"), value)

	// Overwrite first value
	m.SetLoc("key1", nil, dyn.V("qux"))
	assert.Equal(t, 2, m.Len())

	// Confirm the value
	value, ok = m.Get(dyn.V("key1"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("qux"), value)
}

func TestMappingKeysValues(t *testing.T) {
	// Configure mapping
	var m dyn.Mapping
	m.SetLoc("key1", nil, dyn.V("foo"))
	m.SetLoc("key2", nil, dyn.V("bar"))

	// Confirm keys
	keys := m.Keys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, dyn.V("key1"))
	assert.Contains(t, keys, dyn.V("key2"))

	// Confirm values
	values := m.Values()
	assert.Len(t, values, 2)
	assert.Contains(t, values, dyn.V("foo"))
	assert.Contains(t, values, dyn.V("bar"))
}

func TestMappingClone(t *testing.T) {
	// Configure mapping
	var m1 dyn.Mapping
	m1.SetLoc("key1", nil, dyn.V("foo"))
	m1.SetLoc("key2", nil, dyn.V("bar"))

	// Clone mapping
	m2 := m1.Clone()
	assert.Equal(t, m1.Len(), m2.Len())

	// Modify original mapping
	m1.SetLoc("key1", nil, dyn.V("qux"))

	// Confirm values
	value, ok := m1.Get(dyn.V("key1"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("qux"), value)
	value, ok = m2.Get(dyn.V("key1"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("foo"), value)
}

func TestMappingMerge(t *testing.T) {
	var m1 dyn.Mapping
	for i := range 10 {
		m1.SetLoc(strconv.Itoa(i), nil, dyn.V(i))
	}

	var m2 dyn.Mapping
	for i := 5; i < 15; i++ {
		m2.SetLoc(strconv.Itoa(i), nil, dyn.V(i))
	}

	var out dyn.Mapping
	out.Merge(m1)
	assert.Equal(t, 10, out.Len())
	out.Merge(m2)
	assert.Equal(t, 15, out.Len())
}
