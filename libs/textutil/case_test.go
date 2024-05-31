package textutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCamelToSnakeCase(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "test",
			expected: "test",
		},
		{
			input:    "testTest",
			expected: "test_test",
		},
		{
			input:    "testTestTest",
			expected: "test_test_test",
		},
		{
			input:    "TestTest",
			expected: "test_test",
		},
		{
			input:    "TestTestTest",
			expected: "test_test_test",
		},
	}

	for _, c := range cases {
		output := CamelToSnakeCase(c.input)
		assert.Equal(t, c.expected, output)
	}
}
