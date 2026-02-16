package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

type postgresBranchConverter struct{}

func (c postgresBranchConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// The bundle config has flattened BranchSpec fields at the top level.
	// Terraform expects them nested in a "spec" block.
	specFields := specFieldNames(schema.ResourcePostgresBranchSpec{})
	topLevelFields := []string{"branch_id", "parent"}

	// Build the spec block from the flattened fields
	specMap := make(map[string]dyn.Value)
	for _, field := range specFields {
		if v := vin.Get(field); v.Kind() != dyn.KindInvalid {
			specMap[field] = v
		}
	}

	// Build the output with top-level fields and spec
	outMap := make(map[string]dyn.Value)

	// Keep top-level fields
	for _, field := range topLevelFields {
		if v := vin.Get(field); v.Kind() != dyn.KindInvalid {
			outMap[field] = v
		}
	}

	// Add spec block if we have any spec fields
	if len(specMap) > 0 {
		outMap["spec"] = dyn.V(specMap)
	}

	vout := dyn.V(outMap)

	// Normalize the output value to the Terraform schema.
	vout, diags := convert.Normalize(schema.ResourcePostgresBranch{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "postgres branch normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	out.PostgresBranch[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("postgres_branches", "databricks_postgres_branch", postgresBranchConverter{})
}
