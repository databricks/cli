package main

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
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
			EnumLaunchStages: map[string]annotation.LaunchStage{
				"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW",
			},
		})
		assert.Equal(t, "Type of endpoint.", s.Description)
		assert.Equal(t, []string{"[Public Preview]", ""}, s.EnumDescriptions)
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
			map[string]annotation.LaunchStage{"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW"},
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
			map[string]annotation.LaunchStage{"STORAGE_OPTIMIZED": "PUBLIC_BETA"},
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
			map[string]annotation.LaunchStage{"STORAGE_OPTIMIZED": "GA"},
			nil,
		))
	})

	t.Run("non-string enum entries leave an empty slot", func(t *testing.T) {
		got := buildEnumDescriptions(
			[]any{"A", 42, "B"},
			map[string]annotation.LaunchStage{"A": "PUBLIC_PREVIEW", "B": "PUBLIC_BETA"},
			nil,
		)
		assert.Equal(t, []string{"[Public Preview]", "", "[Beta]"}, got)
	})
}
