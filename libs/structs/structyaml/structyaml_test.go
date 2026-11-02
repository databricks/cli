package structyaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

type testStruct struct {
	Name            string            `json:"name"`
	Map             map[string]string `json:"map"`
	List            []string          `json:"list"`
	LongNameField   string            `json:"long_name_field"`
	ForceSendFields []string          `json:"-"`
	Format          string            `json:"format"`
}

func newTestStruct() *testStruct {
	return &testStruct{
		Name:            "test",
		Map:             map[string]string{"key2": "value2", "key1": "value1"},
		List:            []string{"a", "b", "c"},
		ForceSendFields: []string{"Name"},
		LongNameField:   "long name goes here",
	}
}

func TestStruct(t *testing.T) {
	result, err := Struct(newTestStruct(), "format")
	require.NoError(t, err)
	assert.Equal(t, M(
		"list", []any{"a", "b", "c"},
		"long_name_field", "long name goes here",
		"map", M("key1", "value1", "key2", "value2"),
		"name", "test",
	), result)
}

func TestStructWithOrder(t *testing.T) {
	result, err := Struct(newTestStruct(), "format")
	require.NoError(t, err)
	assert.Equal(t, M(
		"list", []any{"a", "b", "c"},
		"name", "test",
		"map", M("key1", "value1", "key2", "value2"),
		"long_name_field", "long name goes here",
	), result.Order("list", "name", "map"))
}

func TestStructNotAMap(t *testing.T) {
	_, err := Struct([]string{"a"})
	assert.ErrorContains(t, err, "expected map")
}

func TestOrderUnknownKeysKeepOrder(t *testing.T) {
	m := M("d", 1, "c", 2, "b", 3, "a", 4)
	assert.Equal(t, M("a", 4, "b", 3, "d", 1, "c", 2), m.Order("a", "x", "b"))
}

func TestValueKeepsFieldOrder(t *testing.T) {
	type inner struct {
		Zeta  string `json:"zeta"`
		Alpha string `json:"alpha"`
	}
	assert.Equal(t, []any{M("zeta", "z", "alpha", "a")}, Value([]inner{{Zeta: "z", Alpha: "a"}}))
	assert.Nil(t, Value([]inner(nil)))
}

func TestNodeScalars(t *testing.T) {
	tests := []struct {
		value any
		want  string
		style yaml.Style
	}{
		{nil, "null", 0},
		{1, "1", 0},
		{int64(291), "291", 0},
		{1.0, "1", 0},
		{true, "true", 0},
		{"value", "value", 0},
		{"0x123", "0x123", yaml.DoubleQuotedStyle},
		{"0b101", "0b101", yaml.DoubleQuotedStyle},
		{"0123", "0123", yaml.DoubleQuotedStyle},
		{"1.0", "1.0", yaml.DoubleQuotedStyle},
		{"true", "true", yaml.DoubleQuotedStyle},
		{"", "", yaml.DoubleQuotedStyle},
	}
	for _, tt := range tests {
		n := Node(tt.value, nil)
		assert.Equal(t, yaml.ScalarNode, n.Kind)
		assert.Equal(t, tt.want, n.Value)
		assert.Equal(t, tt.style, n.Style, "%v", tt.value)
	}
}

func TestNodeSequence(t *testing.T) {
	for _, v := range []any{[]any{"value1", "value2"}, []string{"value1", "value2"}} {
		n := Node(v, nil)
		assert.Equal(t, yaml.SequenceNode, n.Kind)
		assert.Equal(t, "value1", n.Content[0].Value)
		assert.Equal(t, "value2", n.Content[1].Value)
	}
}

func TestNodeMapKeepsOrder(t *testing.T) {
	n := Node(M("key3", "value3", "key1", "value1"), nil)
	assert.Equal(t, yaml.MappingNode, n.Kind)
	assert.Equal(t, "key3", n.Content[0].Value)
	assert.Equal(t, "value3", n.Content[1].Value)
	assert.Equal(t, "key1", n.Content[2].Value)
	assert.Equal(t, "value1", n.Content[3].Value)
}

func TestNodeGoMapSortsKeys(t *testing.T) {
	n := Node(map[string]any{"b": 1, "a": M("y", 1, "x", 2)}, nil)
	assert.Equal(t, "a", n.Content[0].Value)
	assert.Equal(t, "y", n.Content[1].Content[0].Value)
	assert.Equal(t, "b", n.Content[2].Value)
}

func TestNodeStyle(t *testing.T) {
	styles := map[string]yaml.Style{"styled": yaml.DoubleQuotedStyle}
	n := Node(M(
		"styled", M("key1", "value1", "key2", "value2"),
		"unstyled", M("key3", "value3"),
	), styles)

	styled := n.Content[1]
	assert.Equal(t, yaml.MappingNode, styled.Kind)
	for _, c := range styled.Content {
		assert.Equal(t, yaml.DoubleQuotedStyle, c.Style)
	}

	unstyled := n.Content[3]
	assert.Equal(t, yaml.MappingNode, unstyled.Kind)
	for _, c := range unstyled.Content {
		assert.Equal(t, yaml.Style(0), c.Style)
	}
}

func TestSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dir", "out.yml")
	require.NoError(t, Save(path, M("b", "1", "a", M("c", []any{1, true})), false, nil))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "b: \"1\"\na:\n  c:\n    - 1\n    - true\n", string(data))

	err = Save(path, M("a", 1), false, nil)
	assert.ErrorContains(t, err, "already exists. Use --force to overwrite")

	require.NoError(t, Save(path, M("a", 1), true, nil))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a: 1\n", string(data))

	err = Save(filepath.Dir(path), M("a", 1), true, nil)
	assert.ErrorContains(t, err, "is a directory")
}
