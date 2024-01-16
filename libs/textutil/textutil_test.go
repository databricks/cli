package textutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
