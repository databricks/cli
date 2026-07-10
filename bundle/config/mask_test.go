package config_test

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskSensitiveFieldsNoOp(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{},
	}
	v, err := convert.FromTyped(root, dyn.NilValue)
	require.NoError(t, err)

	masked, err := config.MaskSensitiveFields(v)
	require.NoError(t, err)
	assert.Equal(t, v, masked)
}

// testSensitiveResource mimics a resource type with a sensitive field, used to
// verify SensitiveFieldNames without relying on resources.Secret (which lives
// on a different branch).
type testSensitiveResource struct {
	Name  string `json:"name"`
	Token string `json:"token" bundle:"sensitive"`
}

func TestSensitiveFieldNamesReadsTag(t *testing.T) {
	fields := convert.SensitiveFieldNames(reflect.TypeFor[testSensitiveResource]())
	assert.True(t, fields["token"], "token should be sensitive")
	assert.False(t, fields["name"], "name should not be sensitive")
}

func TestSensitiveFieldNamesNilForNonStruct(t *testing.T) {
	fields := convert.SensitiveFieldNames(reflect.TypeFor[string]())
	assert.Nil(t, fields)
}

func TestSensitiveFieldNamesPointerDereference(t *testing.T) {
	fields := convert.SensitiveFieldNames(reflect.TypeFor[*testSensitiveResource]())
	assert.True(t, fields["token"])
}

// TestMaskSensitiveFieldsOnDynValue tests the masking logic directly on a
// constructed dyn.Value tree, without needing resources.Secret to exist.
func TestMaskSensitiveFieldsOnDynValue(t *testing.T) {
	// Build a minimal dyn.Value that looks like:
	//   resources:
	//     jobs:
	//       my_job:
	//         name: "hello"
	//
	// Since jobs have no sensitive fields, masking should leave it unchanged.
	v := dyn.NewValue(map[string]dyn.Value{
		"resources": dyn.NewValue(map[string]dyn.Value{
			"jobs": dyn.NewValue(map[string]dyn.Value{
				"my_job": dyn.NewValue(map[string]dyn.Value{
					"name": dyn.NewValue("hello", nil),
				}, nil),
			}, nil),
		}, nil),
	}, nil)

	masked, err := config.MaskSensitiveFields(v)
	require.NoError(t, err)

	name, err := dyn.GetByPath(masked, dyn.MustPathFromString("resources.jobs.my_job.name"))
	require.NoError(t, err)
	assert.Equal(t, "hello", name.MustString())
}
