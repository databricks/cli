package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestKeyReturnsNameFromJsonTag(T *testing.T) {
	type test struct {
		Name        string `json:"name"`
		Description string `json:"-"`
		Another     string `json:""`
	}

	v := &test{}
	k, ok := dyn.ConfigKey(v, "Name")
	assert.True(T, ok)
	assert.Equal(T, "name", k)

	k, ok = dyn.ConfigKey(v, "Description")
	assert.False(T, ok)
	assert.Equal(T, "Description", k)

	k, ok = dyn.ConfigKey(v, "Another")
	assert.False(T, ok)
	assert.Equal(T, "Another", k)

	k, ok = dyn.ConfigKey(v, "NotExists")
	assert.False(T, ok)
	assert.Equal(T, "NotExists", k)
}

func TestConvertToMapValue(t *testing.T) {
	type test struct {
		Name            string            `json:"name"`
		Map             map[string]string `json:"map"`
		List            []string          `json:"list"`
		ForceSendFields []string          `json:"-"`
	}

	v := &test{
		Name: "test",
		Map: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		List: []string{"a", "b", "c"},
		ForceSendFields: []string{
			"Name",
		},
	}
	result, err := ConvertToMapValue(v, dyn.NewOrder([]string{}), map[string]dyn.Value{})
	assert.NoError(t, err)

	assert.Equal(t, map[string]dyn.Value{
		"name": dyn.NewValue("test", dyn.Location{Line: 1}),
		"map": dyn.NewValue(map[string]dyn.Value{
			"key1": dyn.V("value1"),
			"key2": dyn.V("value2"),
		}, dyn.Location{Line: 2}),
		"list": dyn.NewValue([]dyn.Value{
			dyn.V("a"),
			dyn.V("b"),
			dyn.V("c"),
		}, dyn.Location{Line: 3}),
	}, result.MustMap())
}

func TestConvertToMapValueWithOrder(t *testing.T) {
	type test struct {
		Name            string            `json:"name"`
		Map             map[string]string `json:"map"`
		List            []string          `json:"list"`
		ForceSendFields []string          `json:"-"`
	}

	v := &test{
		Name: "test",
		Map: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		List: []string{"a", "b", "c"},
		ForceSendFields: []string{
			"Name",
		},
	}
	result, err := ConvertToMapValue(v, dyn.NewOrder([]string{"List", "Name", "Map"}), map[string]dyn.Value{})
	assert.NoError(t, err)

	assert.Equal(t, map[string]dyn.Value{
		"list": dyn.NewValue([]dyn.Value{
			dyn.V("a"),
			dyn.V("b"),
			dyn.V("c"),
		}, dyn.Location{Line: -3}),
		"name": dyn.NewValue("test", dyn.Location{Line: -2}),
		"map": dyn.NewValue(map[string]dyn.Value{
			"key1": dyn.V("value1"),
			"key2": dyn.V("value2"),
		}, dyn.Location{Line: -1}),
	}, result.MustMap())
}

func TestNormaliseString(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "test",
			expected: "test",
		},
		{
			input:    "test test",
			expected: "test_test",
		},
		{
			input:    "test-test",
			expected: "test_test",
		},
		{
			input:    "test_test",
			expected: "test_test",
		},
		{
			input:    "test.test",
			expected: "test_test",
		},
		{
			input:    "test/test",
			expected: "test_test",
		},
		{
			input:    "test/test.test",
			expected: "test_test_test",
		},
		{
			input:    "TestTest",
			expected: "testtest",
		},
		{
			input:    "TestTestTest",
			expected: "testtesttest",
		}}

	for _, c := range cases {
		assert.Equal(t, c.expected, NormaliseString(c.input))
	}
}
