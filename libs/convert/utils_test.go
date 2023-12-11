package convert

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestKeyReturnsNameFromJsonTag(T *testing.T) {
	type test struct {
		Name        string `json:"name"`
		Description string `json:"-"`
		Another     string `json:""`
	}

	v := &test{}
	assert.Equal(T, "name", key(v, "Name"))
	assert.Equal(T, "Description", key(v, "Description"))
	assert.Equal(T, "Another", key(v, "Another"))
	assert.Equal(T, "NotExists", key(v, "NotExists"))
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
	result, err := convertToMapValue(v, newOrder([]string{}), map[string]config.Value{})
	assert.NoError(t, err)

	assert.Equal(t, map[string]config.Value{
		"name": config.NewValue("test", config.Location{Line: 1}),
		"map": config.NewValue(map[string]config.Value{
			"key1": config.V("value1"),
			"key2": config.V("value2"),
		}, config.Location{Line: 2}),
		"list": config.NewValue([]config.Value{
			config.V("a"),
			config.V("b"),
			config.V("c"),
		}, config.Location{Line: 3}),
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
	result, err := convertToMapValue(v, newOrder([]string{"List", "Name", "Map"}), map[string]config.Value{})
	assert.NoError(t, err)

	assert.Equal(t, map[string]config.Value{
		"list": config.NewValue([]config.Value{
			config.V("a"),
			config.V("b"),
			config.V("c"),
		}, config.Location{Line: -3}),
		"name": config.NewValue("test", config.Location{Line: -2}),
		"map": config.NewValue(map[string]config.Value{
			"key1": config.V("value1"),
			"key2": config.V("value2"),
		}, config.Location{Line: -1}),
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
