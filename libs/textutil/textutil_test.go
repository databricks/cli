package textutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeString(t *testing.T) {
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
		},
		{
			input:    ".test//test..test",
			expected: "test_test_test",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, NormalizeString(c.input))
	}
}

func TestNormalizePathComponent(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// Basic cases
		{
			input:    "test",
			expected: "test",
		},
		{
			input:    "test-test",
			expected: "test-test",
		},
		{
			input:    "test_test",
			expected: "test_test",
		},
		{
			input:    "test.test",
			expected: "test.test",
		},
		{
			input:    "file with spaces.txt",
			expected: "file with spaces.txt",
		},
		{
			input:    "MixedCase File.txt",
			expected: "MixedCase File.txt",
		},
		{
			input:    "UPPERCASE_FILE.TXT",
			expected: "UPPERCASE_FILE.TXT",
		},
		{
			input:    "lowercase_file.txt",
			expected: "lowercase_file.txt",
		},
		// Windows-incompatible characters
		{
			input:    "test/test",
			expected: "test_test",
		},
		{
			input:    "test/test.test",
			expected: "test_test.test",
		},
		{
			input:    ".test//test..test",
			expected: ".test__test..test",
		},
		{
			input:    "Test DLT: First Test",
			expected: "Test DLT_ First Test",
		},
		{
			input:    "Test DLT: First Test with multiple: colons and spaces",
			expected: "Test DLT_ First Test with multiple_ colons and spaces",
		},
		{
			input:    "Test DLT: First Test with special chars: < > : \" | ? * \\ /",
			expected: "Test DLT_ First Test with special chars_ _ _ _ _ _ _ _ _ _",
		},
		{
			input:    "file with <incompatible> chars",
			expected: "file with _incompatible_ chars",
		},
		// Reserved Windows filenames
		{
			input:    "CON",
			expected: "CON_",
		},
		{
			input:    "COM1",
			expected: "COM1_",
		},
		{
			input:    "LPT1",
			expected: "LPT1_",
		},
		{
			input:    "CON.txt",
			expected: "CON_.txt",
		},
		// Case-insensitive reserved names
		{
			input:    "con",
			expected: "con_",
		},
		{
			input:    "CON.TXT",
			expected: "CON_.TXT",
		},
		// Trailing spaces and periods
		{
			input:    "test ",
			expected: "test",
		},
		{
			input:    "test  ",
			expected: "test",
		},
		{
			input:    "test.",
			expected: "test",
		},
		{
			input:    "test..",
			expected: "test",
		},
		{
			input:    "test .",
			expected: "test",
		},
		{
			input:    "test . ",
			expected: "test",
		},
		{
			input:    "test.txt ",
			expected: "test.txt",
		},
		{
			input:    "test.txt.",
			expected: "test.txt",
		},
		{
			input:    "test.txt .",
			expected: "test.txt",
		},
		// Empty and invalid cases
		{
			input:    "",
			expected: "untitled",
		},
		{
			input:    ":::",
			expected: "___",
		},
		{
			input:    "***",
			expected: "___",
		},
		{
			input:    "   ",
			expected: "untitled",
		},
		{
			input:    "...",
			expected: "untitled",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, NormalizePathComponent(c.input))
	}
}

func TestNormalizePathComponentLengthLimit(t *testing.T) {
	// Test that very long filenames are truncated
	longName := strings.Repeat("a", 300)
	result := NormalizePathComponent(longName)
	assert.Len(t, result, 255)
	assert.True(t, strings.HasSuffix(result, "a"))

	// Test that truncation works correctly for long names
	longNameWithSpaces := strings.Repeat("a ", 150)
	result = NormalizePathComponent(longNameWithSpaces)
	assert.Len(t, result, 255)
}
