package jsonsaver

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

func TestMarshalString(t *testing.T) {
	b, err := Marshal(dyn.V("string"))
	require.NoError(t, err)
	require.JSONEq(t, `"string"`, string(b))
}

func TestMarshalBool(t *testing.T) {
	b, err := Marshal(dyn.V(true))
	require.NoError(t, err)
	require.JSONEq(t, `true`, string(b))
}

func TestMarshalInt(t *testing.T) {
	b, err := Marshal(dyn.V(42))
	require.NoError(t, err)
	require.JSONEq(t, `42`, string(b))
}

func TestMarshalFloat(t *testing.T) {
	b, err := Marshal(dyn.V(42.1))
	require.NoError(t, err)
	require.JSONEq(t, `42.1`, string(b))
}

func TestMarshalTime(t *testing.T) {
	b, err := Marshal(dyn.V(dyn.MustTime("2021-01-01T00:00:00Z")))
	require.NoError(t, err)
	require.JSONEq(t, `"2021-01-01T00:00:00Z"`, string(b))
}

func TestMarshalMap(t *testing.T) {
	m := dyn.NewMapping()
	m.Set(dyn.V("key1"), dyn.V("value1"))
	m.Set(dyn.V("key2"), dyn.V("value2"))

	b, err := Marshal(dyn.V(m))
	require.NoError(t, err)
	require.JSONEq(t, `{"key1":"value1","key2":"value2"}`, string(b))
}

func TestMarshalSequence(t *testing.T) {
	var s []dyn.Value
	s = append(s, dyn.V("value1"))
	s = append(s, dyn.V("value2"))

	b, err := Marshal(dyn.V(s))
	require.NoError(t, err)
	require.JSONEq(t, `["value1","value2"]`, string(b))
}

func TestMarshalComplex(t *testing.T) {
	map1 := dyn.NewMapping()
	map1.Set(dyn.V("str1"), dyn.V("value1"))
	map1.Set(dyn.V("str2"), dyn.V("value2"))

	seq1 := []dyn.Value{}
	seq1 = append(seq1, dyn.V("value1"))
	seq1 = append(seq1, dyn.V("value2"))

	root := dyn.NewMapping()
	root.Set(dyn.V("map1"), dyn.V(map1))
	root.Set(dyn.V("seq1"), dyn.V(seq1))

	// Marshal without indent.
	b, err := Marshal(dyn.V(root))
	require.NoError(t, err)
	require.Equal(t, `{"map1":{"str1":"value1","str2":"value2"},"seq1":["value1","value2"]}`+"\n", string(b))

	// Marshal with indent.
	b, err = MarshalIndent(dyn.V(root), "", "  ")
	require.NoError(t, err)
	require.Equal(t, `{
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
