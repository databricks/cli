package main

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

const testTypePath = "github.com/databricks/cli/bundle/config.Foo"

func TestDropShadowingPlaceholders(t *testing.T) {
	tests := []struct {
		name      string
		fromFile  annotation.File
		extracted annotation.File
		want      annotation.File
	}{
		{
			name: "placeholder shadowing an upstream description is dropped",
			fromFile: annotation.File{
				testTypePath: {"field": {Description: annotation.Placeholder}},
			},
			extracted: annotation.File{
				testTypePath: {"field": {Description: "upstream description"}},
			},
			want: annotation.File{},
		},
		{
			name: "placeholder without upstream description is kept",
			fromFile: annotation.File{
				testTypePath: {"field": {Description: annotation.Placeholder}},
			},
			extracted: annotation.File{},
			want: annotation.File{
				testTypePath: {"field": {Description: annotation.Placeholder}},
			},
		},
		{
			name: "placeholder with other fields loses only the placeholder",
			fromFile: annotation.File{
				testTypePath: {"field": {Description: annotation.Placeholder, DeprecationMessage: "deprecated"}},
			},
			extracted: annotation.File{
				testTypePath: {"field": {Description: "upstream description"}},
			},
			want: annotation.File{
				testTypePath: {"field": {DeprecationMessage: "deprecated"}},
			},
		},
		{
			name: "real description override is untouched",
			fromFile: annotation.File{
				testTypePath: {"field": {Description: "hand-written override"}},
			},
			extracted: annotation.File{
				testTypePath: {"field": {Description: "upstream description"}},
			},
			want: annotation.File{
				testTypePath: {"field": {Description: "hand-written override"}},
			},
		},
		{
			name: "other fields of the same type are kept",
			fromFile: annotation.File{
				testTypePath: {
					"stale": {Description: annotation.Placeholder},
					"todo":  {Description: annotation.Placeholder},
				},
			},
			extracted: annotation.File{
				testTypePath: {"stale": {Description: "upstream description"}},
			},
			want: annotation.File{
				testTypePath: {"todo": {Description: annotation.Placeholder}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dropShadowingPlaceholders(test.fromFile, test.extracted)
			assert.Equal(t, test.want, test.fromFile)
		})
	}
}

// A stale placeholder must not swallow the upstream description in the merged
// view the schema is generated from.
func TestNewAnnotationHandlerResolvesShadowedDescriptions(t *testing.T) {
	extracted := annotation.File{
		testTypePath: {"field": {Description: "upstream description"}},
	}
	fromFile := annotation.File{
		testTypePath: {"field": {Description: annotation.Placeholder, DeprecationMessage: "deprecated"}},
	}

	h, err := newAnnotationHandler(extracted, fromFile)
	require.NoError(t, err)

	merged := h.parsedAnnotations[testTypePath]["field"]
	assert.Equal(t, "upstream description", merged.Description)
	assert.Equal(t, "deprecated", merged.DeprecationMessage)
}
