package dyn_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
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
	err := m.Set(dyn.V("key"), dyn.V("value"))
	assert.NoError(t, err)
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

func TestMappingSet(t *testing.T) {
	var err error
	var m dyn.Mapping

	// Set a value
	err = m.Set(dyn.V("key1"), dyn.V("foo"))
	assert.NoError(t, err)
	assert.Equal(t, 1, m.Len())

	// Confirm the value
	value, ok := m.GetByString("key1")
	assert.True(t, ok)
	assert.Equal(t, dyn.V("foo"), value)

	// Set another value
	err = m.Set(dyn.V("key2"), dyn.V("bar"))
	assert.NoError(t, err)
	assert.Equal(t, 2, m.Len())

	// Confirm the value
	value, ok = m.Get(dyn.V("key2"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("bar"), value)

	// Overwrite first value
	err = m.Set(dyn.V("key1"), dyn.V("qux"))
	assert.NoError(t, err)
	assert.Equal(t, 2, m.Len())

	// Confirm the value
	value, ok = m.Get(dyn.V("key1"))
	assert.True(t, ok)
	assert.Equal(t, dyn.V("qux"), value)

	// Try to set non-string key
	err = m.Set(dyn.V(1), dyn.V("qux"))
	assert.Error(t, err)
	assert.Equal(t, 2, m.Len())
}

func TestMappingKeysValues(t *testing.T) {
	var err error

	// Configure mapping
	var m dyn.Mapping
	err = m.Set(dyn.V("key1"), dyn.V("foo"))
	assert.NoError(t, err)
	err = m.Set(dyn.V("key2"), dyn.V("bar"))
	assert.NoError(t, err)

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
	var err error

	// Configure mapping
	var m1 dyn.Mapping
	err = m1.Set(dyn.V("key1"), dyn.V("foo"))
	assert.NoError(t, err)
	err = m1.Set(dyn.V("key2"), dyn.V("bar"))
	assert.NoError(t, err)

	// Clone mapping
	m2 := m1.Clone()
	assert.Equal(t, m1.Len(), m2.Len())

	// Modify original mapping
	err = m1.Set(dyn.V("key1"), dyn.V("qux"))
	assert.NoError(t, err)

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
	for i := 0; i < 10; i++ {
		err := m1.Set(dyn.V(fmt.Sprintf("%d", i)), dyn.V(i))
		require.NoError(t, err)
	}

	var m2 dyn.Mapping
	for i := 5; i < 15; i++ {
		err := m2.Set(dyn.V(fmt.Sprintf("%d", i)), dyn.V(i))
		require.NoError(t, err)
	}

	var out dyn.Mapping
	out.Merge(m1)
	assert.Equal(t, 10, out.Len())
	out.Merge(m2)
	assert.Equal(t, 15, out.Len())
}
