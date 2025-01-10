package main

import (
	"testing"
)

func TestConvertLinksToAbsoluteUrl(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// {
		// 	input:    "",
		// 	expected: "",
		// },
		// {
		// 	input:    "Some text (not a link)",
		// 	expected: "Some text (not a link)",
		// },
		// {
		// 	input:    "This is a link to [_](#section)",
		// 	expected: "This is a link to [section](https://docs.databricks.com/dev-tools/bundles/reference.html#section)",
		// },
		// {
		// 	input:    "This is a link to [_](/dev-tools/bundles/resources.html#dashboard)",
		// 	expected: "This is a link to [dashboard](https://docs.databricks.com/dev-tools/bundles/resources.html#dashboard)",
		// },
		// {
		// 	input:    "This is a link to [_](/dev-tools/bundles/resources.html)",
		// 	expected: "This is a link to [link](https://docs.databricks.com/dev-tools/bundles/resources.html)",
		// },
		// {
		// 	input:    "This is a link to [external](https://external.com)",
		// 	expected: "This is a link to [external](https://external.com)",
		// },
		// {
		// 	input:    "This is a link to [pipelines](/api/workspace/pipelines/create)",
		// 	expected: "This is a link to [pipelines](https://docs.databricks.com/api/workspace/pipelines/create)",
		// },
		{
			input:    "The registered model resource allows you to define models in \u003cUC\u003e. For information about \u003cUC\u003e [registered models](/api/workspace/registeredmodels/create), [registered models 2](/api/workspace/registeredmodels/create)",
			expected: "The registered model resource allows you to define models in \u003cUC\u003e. For information about \u003cUC\u003e [registered models](/api/workspace/registeredmodels/create), [registered models 2](/api/workspace/registeredmodels/create)",
		},
	}

	for _, test := range tests {
		result := convertLinksToAbsoluteUrl(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s', but got '%s'", test.input, test.expected, result)
		}
	}
}
