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

func TestDropPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		fromFile annotation.File
		want     annotation.File
	}{
		{
			name: "placeholder-only entry is dropped with its type",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder}}},
			},
			want: annotation.File{},
		},
		{
			name: "placeholder with other fields loses only the placeholder",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder, DeprecationMessage: "deprecated"}}},
			},
			want: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {DeprecationMessage: "deprecated"}}},
			},
		},
		{
			name: "real description is untouched",
			fromFile: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "hand-written override"}}},
			},
			want: annotation.File{
				testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "hand-written override"}}},
			},
		},
		{
			name: "type docs keep the entry alive",
			fromFile: annotation.File{
				testTypePath: {
					Self:   annotation.Descriptor{Description: "the type"},
					Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder}},
				},
			},
			want: annotation.File{
				testTypePath: {Self: annotation.Descriptor{Description: "the type"}, Fields: map[string]annotation.Descriptor{}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dropPlaceholders(test.fromFile)
			assert.Equal(t, test.want, test.fromFile)
		})
	}
}

func TestSplitOverrides(t *testing.T) {
	extracted := annotation.File{
		testTypePath: {
			Self: annotation.Descriptor{Description: "upstream type"},
			Fields: map[string]annotation.Descriptor{
				"documented":   {Description: "upstream description"},
				"undocumented": {},
				"deprecated":   {Description: "upstream description"},
			},
		},
	}
	fromFile := annotation.File{
		testTypePath: {
			Self: annotation.Descriptor{MarkdownDescription: "docs-site blurb"},
			Fields: map[string]annotation.Descriptor{
				"documented":   {Description: "hand-written override"},
				"undocumented": {Description: "gap fill"},
				"cli_only":     {Description: "cli field"},
				"deprecated":   {DeprecationMessage: "gone"},
			},
		},
		"github.com/databricks/cli/bundle/config.Bar": {
			Self: annotation.Descriptor{Description: "cli-only type"},
		},
	}

	cliOnly, overrides := splitOverrides(fromFile, extracted)

	assert.Equal(t, annotation.File{
		testTypePath: {
			Self: annotation.Descriptor{MarkdownDescription: "docs-site blurb"},
			Fields: map[string]annotation.Descriptor{
				"cli_only": {Description: "cli field"},
			},
		},
		"github.com/databricks/cli/bundle/config.Bar": {
			Self: annotation.Descriptor{Description: "cli-only type"},
		},
	}, cliOnly)
	assert.Equal(t, annotation.File{
		testTypePath: {
			Fields: map[string]annotation.Descriptor{
				"documented":   {Description: "hand-written override"},
				"undocumented": {Description: "gap fill"},
				"deprecated":   {DeprecationMessage: "gone"},
			},
		},
	}, overrides)
}

// A placeholder left in the annotations file must not swallow the upstream
// description in the merged view the schema is generated from.
func TestPlaceholderDoesNotShadowMergedDescription(t *testing.T) {
	extracted := annotation.File{
		testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: "upstream description"}}},
	}
	fromFile := annotation.File{
		testTypePath: {Fields: map[string]annotation.Descriptor{"field": {Description: annotation.Placeholder, DeprecationMessage: "deprecated"}}},
	}

	dropPlaceholders(fromFile)
	h, err := newAnnotationHandler(extracted, fromFile)
	require.NoError(t, err)

	merged := h.parsedAnnotations[testTypePath].Fields["field"]
	assert.Equal(t, "upstream description", merged.Description)
	assert.Equal(t, "deprecated", merged.DeprecationMessage)
}

func TestAssignAnnotationLaunchStage(t *testing.T) {
	// Each stamped stage emits x-databricks-launch-stage and prefixes the
	// description with its tag (GA renders no tag); only private preview also
	// hides the field from autocomplete.
	tests := []struct {
		name         string
		stage        clijson.LaunchStage
		wantDesc     string
		wantSuppress bool
	}{
		{"private preview", clijson.LaunchStagePrivatePreview, "[Private Preview] A field.", true},
		{"public beta", clijson.LaunchStagePublicBeta, "[Beta] A field.", false},
		{"public preview", clijson.LaunchStagePublicPreview, "[Public Preview] A field.", false},
		{"GA", clijson.LaunchStageGA, "A field.", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &jsonschema.Schema{}
			assignAnnotation(s, annotation.Descriptor{Description: "A field.", LaunchStage: tc.stage})
			assert.Equal(t, tc.wantDesc, s.Description)
			assert.Equal(t, tc.wantSuppress, s.DoNotSuggest)
			assert.Equal(t, string(tc.stage), s.LaunchStage)
		})
	}

	t.Run("unstamped field emits no stage", func(t *testing.T) {
		s := &jsonschema.Schema{}
		assignAnnotation(s, annotation.Descriptor{Description: "A field."})
		assert.Equal(t, "A field.", s.Description)
		assert.Empty(t, s.LaunchStage)
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
		got := buildEnumDescriptions(
			enum,
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
		got := buildEnumDescriptions(
			enum,
			map[string]clijson.LaunchStage{"STORAGE_OPTIMIZED": "PUBLIC_BETA"},
			nil,
		)
		assert.Equal(t, []string{"[Beta]", ""}, got)
	})

	t.Run("description only is preserved verbatim", func(t *testing.T) {
		got := buildEnumDescriptions(
			enum,
			nil,
			map[string]string{"STORAGE_OPTIMIZED": "Storage-optimized endpoint."},
		)
		assert.Equal(t, []string{"Storage-optimized endpoint.", ""}, got)
	})

	t.Run("returns nil when neither stage nor description has content", func(t *testing.T) {
		assert.Nil(t, buildEnumDescriptions(enum, nil, nil))
		assert.Nil(t, buildEnumDescriptions(
			enum,
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
