package tnresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/structwalk"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestNewJobResource(t *testing.T) {
	client := &databricks.WorkspaceClient{}

	cfg := &resources.Job{
		JobSettings: jobs.JobSettings{
			Name: "test-job",
		},
	}

	res, cfgType, err := New(client, "jobs", "test-job", cfg)
	require.NoError(t, err)
	require.NotNil(t, res)

	// Ensure we received the correct resource type.
	require.IsType(t, &ResourceJob{}, res)
	require.IsType(t, reflect.TypeOf(ResourceJob{}.config), cfgType)
	require.IsType(t, reflect.TypeOf(jobs.JobSettings{}), cfgType)

	// The underlying config should match what we passed in.
	r := res.(*ResourceJob)
	require.Equal(t, cfg.JobSettings, r.config)
}

// validateFields uses structwalk to generate all valid field paths and checks membership.
func validateFields(t *testing.T, configType reflect.Type, fields map[string]struct{}) {
	validPaths := make(map[string]struct{})

	err := structwalk.WalkType(configType, func(path *structpath.PathNode, typ reflect.Type) bool {
		validPaths[path.String()] = struct{}{}
		return true // continue walking
	})
	require.NoError(t, err)

	for fieldPath := range fields {
		if _, exists := validPaths[fieldPath]; !exists {
			t.Errorf("invalid field '%s' for %s", fieldPath, configType)
		}
	}
}

// TestRecreateFieldsValidation validates that all fields in RecreateFields
// exist in the corresponding ConfigType for each resource.
func TestRecreateFieldsValidation(t *testing.T) {
	for resourceName, settings := range SupportedResources {
		if len(settings.RecreateFields) == 0 {
			continue
		}
		t.Run(resourceName, func(t *testing.T) {
			validateFields(t, settings.ConfigType, settings.RecreateFields)
		})
	}
}
