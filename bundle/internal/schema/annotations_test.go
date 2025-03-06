package main

import (
	"testing"
)

func TestConvertLinksToAbsoluteUrl(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    "Some text (not a link)",
			expected: "Some text (not a link)",
		},
		{
			input:    "This is a link to [_](#section)",
			expected: "This is a link to [section](https://docs.databricks.com/dev-tools/bundles/reference.html#section)",
		},
		{
			input:    "This is a link to [_](/dev-tools/bundles/resources.html#dashboard)",
			expected: "This is a link to [dashboard](https://docs.databricks.com/dev-tools/bundles/resources.html#dashboard)",
		},
		{
			input:    "This is a link to [_](/dev-tools/bundles/resources.html)",
			expected: "This is a link to [link](https://docs.databricks.com/dev-tools/bundles/resources.html)",
		},
		{
			input:    "This is a link to [external](https://external.com)",
			expected: "This is a link to [external](https://external.com)",
		},
		{
			input:    "This is a link to [one](/relative), [two](/relative-2)",
			expected: "This is a link to [one](https://docs.databricks.com/relative), [two](https://docs.databricks.com/relative-2)",
		},
	}

	for _, test := range tests {
		result := convertLinksToAbsoluteUrl(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s', but got '%s'", test.input, test.expected, result)
		}
	}
}
