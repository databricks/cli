package jsonsaver

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestMarshal_String(t *testing.T) {
	b, err := Marshal(dyn.V("string"))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `"string"`, string(b))
	}
}

func TestMarshal_Bool(t *testing.T) {
	b, err := Marshal(dyn.V(true))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `true`, string(b))
	}
}

func TestMarshal_Int(t *testing.T) {
	b, err := Marshal(dyn.V(42))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `42`, string(b))
	}
}

func TestMarshal_Float(t *testing.T) {
	b, err := Marshal(dyn.V(42.1))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `42.1`, string(b))
	}
}

func TestMarshal_Time(t *testing.T) {
	b, err := Marshal(dyn.V(dyn.MustTime("2021-01-01T00:00:00Z")))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `"2021-01-01T00:00:00Z"`, string(b))
	}
}

func TestMarshal_Map(t *testing.T) {
	m := dyn.NewMapping()
	require.NoError(t, m.Set(dyn.V("key1"), dyn.V("value1")))
	require.NoError(t, m.Set(dyn.V("key2"), dyn.V("value2")))

	b, err := Marshal(dyn.V(m))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `{"key1":"value1","key2":"value2"}`, string(b))
	}
}

func TestMarshal_Sequence(t *testing.T) {
	var s []dyn.Value
	s = append(s, dyn.V("value1"))
	s = append(s, dyn.V("value2"))

	b, err := Marshal(dyn.V(s))
	if assert.NoError(t, err) {
		assert.JSONEq(t, `["value1","value2"]`, string(b))
	}
}

func TestMarshal_Complex(t *testing.T) {
	map1 := dyn.NewMapping()
	require.NoError(t, map1.Set(dyn.V("str1"), dyn.V("value1")))
	require.NoError(t, map1.Set(dyn.V("str2"), dyn.V("value2")))

	seq1 := []dyn.Value{}
	seq1 = append(seq1, dyn.V("value1"))
	seq1 = append(seq1, dyn.V("value2"))

	root := dyn.NewMapping()
	require.NoError(t, root.Set(dyn.V("map1"), dyn.V(map1)))
	require.NoError(t, root.Set(dyn.V("seq1"), dyn.V(seq1)))

	// Marshal without indent.
	b, err := Marshal(dyn.V(root))
	if assert.NoError(t, err) {
		assert.Equal(t, `{"map1":{"str1":"value1","str2":"value2"},"seq1":["value1","value2"]}`+"\n", string(b))
	}

	// Marshal with indent.
	b, err = MarshalIndent(dyn.V(root), "", "  ")
	if assert.NoError(t, err) {
		assert.Equal(t, `{
  "map1": {
    "str1": "value1",
    "str2": "value2"
  },
  "seq1": [
    "value1",
    "value2"
  ]
}`+"\n", string(b))
	}
}
