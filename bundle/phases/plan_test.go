package phases

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

func TestCheckPreventDestroyForAllResources(t *testing.T) {
	supportedResources := config.SupportedResources()

	for resourceType := range supportedResources {
		t.Run(resourceType, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{
						Name: "test",
					},
					Resources: config.Resources{},
				},
			}

			ctx := context.Background()
			bundle.ApplyFuncContext(ctx, b, func(ctx context.Context, b *bundle.Bundle) {
				// Use Mutate to set the configuration dynamically
				err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
					// Set the resource with lifecycle.prevent_destroy = true
					return dyn.Set(v, "resources", dyn.NewValue(map[string]dyn.Value{
						resourceType: dyn.NewValue(map[string]dyn.Value{
							"test_resource": dyn.NewValue(map[string]dyn.Value{
								"lifecycle": dyn.NewValue(map[string]dyn.Value{
									"prevent_destroy": dyn.NewValue(true, nil),
								}, nil),
							}, nil),
						}, nil),
					}, nil))
				})
				require.NoError(t, err)
			})

			actions := []deployplan.Action{
				{
					ResourceNode: deployplan.ResourceNode{
						Group: resourceType,
						Key:   "test_resource",
					},
					ActionType: deployplan.ActionTypeRecreate,
				},
			}

			err := checkForPreventDestroy(b, actions)
			require.Error(t, err)
			require.Contains(t, err.Error(), "resource test_resource has lifecycle.prevent_destroy set")
			require.Contains(t, err.Error(), "but the plan calls for this resource to be recreated or destroyed")
			require.Contains(t, err.Error(), fmt.Sprintf("disable lifecycle.prevent_destroy for %s.test_resource", resourceType))
		})
	}
}
