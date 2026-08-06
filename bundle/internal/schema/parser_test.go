package main

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/internal/clijson"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProjectConfig and testProject mirror the shape of resources.PostgresProject:
// the SDK spec is embedded two levels down, behind an intermediate config struct.
type testProjectConfig struct {
	postgres.ProjectSpec

	ProjectId string `json:"project_id"`
}

type testProject struct {
	testProjectConfig
}

func TestFindRefNestedEmbeddedSDKType(t *testing.T) {
	spec := &clijson.SchemaJSON{Description: "A Lakebase project."}
	p := newParser(map[string]*clijson.SchemaJSON{"postgres.ProjectSpec": spec})

	t.Run("resolves a spec embedded below the first level", func(t *testing.T) {
		got, ok := p.findRef(reflect.TypeFor[testProject]())
		require.True(t, ok)
		assert.Same(t, spec, got)
	})

	t.Run("resolves the SDK type itself", func(t *testing.T) {
		got, ok := p.findRef(reflect.TypeFor[postgres.ProjectSpec]())
		require.True(t, ok)
		assert.Same(t, spec, got)
	})

	t.Run("reports no match when the spec omits the type", func(t *testing.T) {
		_, ok := newParser(nil).findRef(reflect.TypeFor[testProject]())
		assert.False(t, ok)
	})
}

// Descriptions and launch stages from a spec embedded below the first level must
// land on the wrapping resource type, since that is where the schema generator
// flattens the spec's fields to.
func TestExtractAnnotationsNestedEmbeddedSDKType(t *testing.T) {
	p := newParser(map[string]*clijson.SchemaJSON{
		"postgres.ProjectSpec": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"display_name": {
					Description: "Human-readable project name.",
					LaunchStage: "PUBLIC_BETA",
				},
			},
		},
	})

	annotations, err := p.extractAnnotations(reflect.TypeFor[testProject]())
	require.NoError(t, err)

	got := annotations[getPath(reflect.TypeFor[testProject]())].Fields["display_name"]
	assert.Equal(t, "Human-readable project name.", got.Description)
	assert.Equal(t, clijson.LaunchStagePublicBeta, got.LaunchStage)
}

func TestNormalizeLaunchStage(t *testing.T) {
	tests := []struct {
		input string
		want  clijson.LaunchStage
	}{
		{"GA", ""},
		{"", ""},
		{"PUBLIC_PREVIEW", clijson.LaunchStagePublicPreview},
		{"PUBLIC_BETA", clijson.LaunchStagePublicBeta},
		{"PRIVATE_PREVIEW", clijson.LaunchStagePrivatePreview},
	}
	for _, tc := range tests {
		got, err := normalizeLaunchStage(tc.input)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestNormalizeLaunchStageUnknown(t *testing.T) {
	_, err := normalizeLaunchStage("SOMETHING_ELSE")
	assert.Error(t, err)
}

func TestNotableEnumLaunchStages(t *testing.T) {
	t.Run("drops GA, keeps preview values", func(t *testing.T) {
		got, err := notableEnumLaunchStages(map[string]string{
			"STORAGE_OPTIMIZED": "PUBLIC_PREVIEW",
			"STANDARD":          "GA",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]clijson.LaunchStage{"STORAGE_OPTIMIZED": clijson.LaunchStagePublicPreview}, got)
	})

	t.Run("returns nil when every value is GA", func(t *testing.T) {
		got, err := notableEnumLaunchStages(map[string]string{"STANDARD": "GA"})
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		got, err := notableEnumLaunchStages(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("errors on unknown stage", func(t *testing.T) {
		_, err := notableEnumLaunchStages(map[string]string{"X": "SOMETHING_ELSE"})
		assert.Error(t, err)
	})
}

func TestNonEmptyEnumDescriptions(t *testing.T) {
	t.Run("keeps non-empty descriptions", func(t *testing.T) {
		got := nonEmptyEnumDescriptions(map[string]string{
			"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
			"STANDARD":          "Standard endpoint.",
		})
		assert.Equal(t, map[string]string{
			"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
			"STANDARD":          "Standard endpoint.",
		}, got)
	})

	t.Run("drops empty descriptions", func(t *testing.T) {
		got := nonEmptyEnumDescriptions(map[string]string{
			"STORAGE_OPTIMIZED": "Storage-optimized endpoint.",
			"STANDARD":          "",
		})
		assert.Equal(t, map[string]string{"STORAGE_OPTIMIZED": "Storage-optimized endpoint."}, got)
	})

	t.Run("returns nil for empty input", func(t *testing.T) {
		assert.Nil(t, nonEmptyEnumDescriptions(nil))
		assert.Nil(t, nonEmptyEnumDescriptions(map[string]string{"STANDARD": ""}))
	})
}
