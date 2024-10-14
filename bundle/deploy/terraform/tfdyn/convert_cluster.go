package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

func convertClusterResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(compute.ClusterSpec{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "cluster normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type clusterConverter struct{}

func (clusterConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertClusterResource(ctx, vin)
	if err != nil {
		return err
	}

	// We always set no_wait as it allows DABs not to wait for cluster to be started.
	vout, err = dyn.Set(vout, "no_wait", dyn.V(true))
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Cluster[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.ClusterId = fmt.Sprintf("${databricks_cluster.%s.id}", key)
		out.Permissions["cluster_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("clusters", clusterConverter{})
}
