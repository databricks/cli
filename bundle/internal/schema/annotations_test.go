package main

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/cli/libs/jsonschema"
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
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder}}},
			},
			extracted: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "upstream description"}}},
			},
			want: annotation.File{},
		},
		{
			name: "placeholder without upstream description is kept",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder}}},
			},
			extracted: annotation.File{},
			want: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder}}},
			},
		},
		{
			name: "placeholder with other fields loses only the placeholder",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder, DeprecationMessage: "deprecated"}}},
			},
			extracted: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "upstream description"}}},
			},
			want: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {DeprecationMessage: "deprecated"}}},
			},
		},
		{
			name: "real description override is untouched",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "hand-written override"}}},
			},
			extracted: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "upstream description"}}},
			},
			want: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "hand-written override"}}},
			},
		},
		{
			name: "other fields of the same type are kept",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{
					"stale": {Description: annotation.Placeholder},
					"todo":  {Description: annotation.Placeholder},
				}},
			},
			extracted: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"stale": {Description: "upstream description"}}},
			},
			want: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"todo": {Description: annotation.Placeholder}}},
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
func TestStalePlaceholderDoesNotShadowMergedDescription(t *testing.T) {
	extracted := annotation.File{
		testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "upstream description"}}},
	}
	fromFile := annotation.File{
		testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder, DeprecationMessage: "deprecated"}}},
	}

	dropShadowingPlaceholders(fromFile, extracted)
	h, err := newAnnotationHandler(extracted, fromFile)
	require.NoError(t, err)

	merged := h.parsedAnnotations[testTypePath].Fields["field"]
	assert.Equal(t, "upstream description", merged.Description)
	assert.Equal(t, "deprecated", merged.DeprecationMessage)
}

func TestAssignAnnotationLaunchStage(t *testing.T) {
	t.Run("public preview prefixes description and stays suggestible", func(t *testing.T) {
		s := &jsonschema.Schema{}
		assignAnnotation(s, annotation.Descriptor{
			Description: "Target QPS for the endpoint.",
			LaunchStage: "PUBLIC_PREVIEW",
		})
		assert.Equal(t, "[Public Preview] Target QPS for the endpoint.", s.Description)
		assert.False(t, s.DoNotSuggest)
		assert.Empty(t, s.LaunchStage)
	})

	t.Run("public beta prefixes description", func(t *testing.T) {
		s := &jsonschema.Schema{}
		assignAnnotation(s, annotation.Descriptor{
			Description: "A field.",
			LaunchStage: "PUBLIC_BETA",
		})
		assert.Equal(t, "[Beta] A field.", s.Description)
	})

	t.Run("private preview also hides from autocomplete", func(t *testing.T) {
		s := &jsonschema.Schema{}
		// The private-preview stage both prefixes the description and hides the
		// field; it is also emitted as x-databricks-launch-stage for pydabs.
		assignAnnotation(s, annotation.Descriptor{
			Description: "Internal field.",
			LaunchStage: "PRIVATE_PREVIEW",
		})
		assert.Equal(t, "[Private Preview] Internal field.", s.Description)
		assert.True(t, s.DoNotSuggest)
		assert.Equal(t, "PRIVATE_PREVIEW", s.LaunchStage)
	})

	t.Run("per-enum-value launch stages do not leak into description", func(t *testing.T) {
		s := &jsonschema.Schema{}
		assignAnnotation(s, annotation.Descriptor{
			Description: "Type of endpoint.",
			Enum:        []any{"STORAGE_OPTIMIZED", "STANDARD"},
			EnumLaunchStages: map[string]clijson.LaunchStage{
				"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW",
			},
		})
		assert.Equal(t, "Type of endpoint.", s.Description)
		assert.Equal(t, []string{"[Public Preview]", ""}, s.EnumDescriptions)
	})

	t.Run("deduplicates enum values before building enum descriptions", func(t *testing.T) {
		s := &jsonschema.Schema{}
		assignAnnotation(s, annotation.Descriptor{
			Enum: []any{"AWS_SSE_S3", "AWS_SSE_KMS", "AWS_SSE_KMS", "AWS_SSE_S3"},
			EnumDescriptions: map[string]string{
				"AWS_SSE_KMS": "SSE-KMS encryption.",
				"AWS_SSE_S3":  "SSE-S3 encryption.",
			},
		})
		assert.Equal(t, []any{"AWS_SSE_S3", "AWS_SSE_KMS"}, s.Enum)
		assert.Equal(t, []string{"SSE-S3 encryption.", "SSE-KMS encryption."}, s.EnumDescriptions)
	})
}

func TestPrefixWithPreviewTagNoDoubleTag(t *testing.T) {
	t.Run("empty description becomes the tag", func(t *testing.T) {
		assert.Equal(t, "[Public Preview]", prefixWithPreviewTag("", "[Public Preview]"))
	})

	t.Run("normal description is prefixed once", func(t *testing.T) {
		assert.Equal(t, "[Public Preview] A field.", prefixWithPreviewTag("A field.", "[Public Preview]"))
	})

	t.Run("description already starting with the tag is left alone", func(t *testing.T) {
		got := prefixWithPreviewTag("[Public Preview] Already tagged.", "[Public Preview]")
		assert.Equal(t, "[Public Preview] Already tagged.", got)
	})

	t.Run("different tag still prepends even if description has another tag", func(t *testing.T) {
		got := prefixWithPreviewTag("[Private Preview] foo", "[Public Preview]")
		assert.Equal(t, "[Public Preview] [Private Preview] foo", got)
	})
}

func TestBuildEnumDescriptions(t *testing.T) {
	enum := []any{"STORAGE_OPTIMIZED", "STANDARD"}

	t.Run("combines launch stage and description per value", func(t *testing.T) {
		got := buildEnumDescriptions(enum,
			map[string]clijson.LaunchStage{"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW"},
			map[string]string{
				"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
				"STANDARD":          "Standard endpoint.",
			},
		)
		assert.Equal(t, []string{
			"[Public Preview] Storage-optimized endpoint.",
			"Standard endpoint.",
		}, got)
	})

	t.Run("launch stage only emits bracketed label", func(t *testing.T) {
		got := buildEnumDescriptions(enum,
			map[string]clijson.LaunchStage{"STORAGE_OPTIMIZED": "PUBLIC_BETA"},
			nil,
		)
		assert.Equal(t, []string{"[Beta]", ""}, got)
	})

	t.Run("description only is preserved verbatim", func(t *testing.T) {
		got := buildEnumDescriptions(enum,
			nil,
			map[string]string{"STORAGE_OPTIMIZED": "Storage-optimized endpoint."},
		)
		assert.Equal(t, []string{"Storage-optimized endpoint.", ""}, got)
	})

	t.Run("returns nil when neither stage nor description has content", func(t *testing.T) {
		assert.Nil(t, buildEnumDescriptions(enum, nil, nil))
		assert.Nil(t, buildEnumDescriptions(enum,
			map[string]clijson.LaunchStage{"STORAGE_OPTIMIZED": "GA"},
			nil,
		))
	})

	t.Run("non-string enum entries leave an empty slot", func(t *testing.T) {
		got := buildEnumDescriptions(
			[]any{"A", 42, "B"},
			map[string]clijson.LaunchStage{"A": "PUBLIC_PREVIEW", "B": "PUBLIC_BETA"},
			nil,
		)
		assert.Equal(t, []string{"[Public Preview]", "", "[Beta]"}, got)
	})
}
