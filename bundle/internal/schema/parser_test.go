package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/annotation"
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

// TestPostgresResourcesHaveLaunchStageOverride asserts every Postgres resource
// type on config.Resources has a Beta launch-stage override, so an upstream
// stage change cannot silently alter its label (see
// annotation.OverrideLaunchStage). It walks the resource map fields by
// reflection, so a newly added or renamed Postgres resource with no override
// fails here.
func TestPostgresResourcesHaveLaunchStageOverride(t *testing.T) {
	resourcesType := reflect.TypeFor[config.Resources]()
	found := 0
	for _, f := range reflect.VisibleFields(resourcesType) {
		if !strings.HasPrefix(f.Name, "Postgres") {
			continue
		}
		// Each resource is a map[string]*resources.PostgresX; unwrap to the
		// element struct the schema generator keys annotations by.
		elem := f.Type.Elem()
		for elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}
		found++
		typePath := getPath(elem)
		// A configured type returns its override for any input; an unconfigured
		// one returns the input unchanged, so a GA input that survives as GA
		// (rather than becoming Beta) means the override is missing.
		stage := annotation.OverrideLaunchStage(typePath, clijson.LaunchStageGA)
		assert.Equalf(t, clijson.LaunchStagePublicBeta, stage,
			"resource %s (%s) has no launch-stage override; add it to launchStageOverrides in bundle/internal/annotation/preview.go", f.Name, typePath)
	}
	assert.Equal(t, 7, found, "expected 7 Postgres resources; update this test if the set changed")
}

// TestExtractAnnotationsOverridesLaunchStage feeds a GA launch stage for a real
// field of an overridden resource and asserts extractAnnotations rewrites it to
// the override, so an upstream stage change does not alter the label.
// auth_method is a field of postgres.RoleRoleSpec, embedded by
// resources.PostgresRole.
func TestExtractAnnotationsOverridesLaunchStage(t *testing.T) {
	p := newParser(map[string]*clijson.SchemaJSON{
		"postgres.RoleRoleSpec": {
			Fields: map[string]*clijson.SchemaFieldJSON{
				"auth_method": {
					Description: "How the role authenticates.",
					LaunchStage: "GA",
				},
			},
		},
	})

	annotations, err := p.extractAnnotations(reflect.TypeFor[resources.PostgresRole]())
	require.NoError(t, err)

	got := annotations[getPath(reflect.TypeFor[resources.PostgresRole]())].Fields["auth_method"]
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
