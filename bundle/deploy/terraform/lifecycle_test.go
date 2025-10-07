package terraform

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

func TestConvertLifecycleForAllResources(t *testing.T) {
	supportedResources := config.SupportedResources()
	ctx := context.Background()

	for resourceType := range supportedResources {
		t.Run(resourceType, func(t *testing.T) {
			vin := dyn.NewValue(map[string]dyn.Value{
				"resources": dyn.NewValue(map[string]dyn.Value{
					resourceType: dyn.NewValue(map[string]dyn.Value{
						"test_resource": dyn.NewValue(map[string]dyn.Value{
							"lifecycle": dyn.NewValue(map[string]dyn.Value{
								"prevent_destroy": dyn.NewValue(true, nil),
							}, nil),
						}, nil),
					}, nil),
				}, nil),
			}, nil)

			tfroot, err := BundleToTerraformWithDynValue(ctx, vin)
			require.NoError(t, err)

			bytes, err := json.Marshal(tfroot.Resource)
			require.NoError(t, err)
			require.Contains(t, string(bytes), `"lifecycle":{"prevent_destroy":true}`)
		})
	}
}
