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
	// Terraform expects them nested in a "spec" block. We need to restructure:
	// - Keep branch_id, parent at top level
	// - Move expire_time, is_protected, no_expiry, source_branch, source_branch_lsn, source_branch_time, ttl into spec

	specFields := []string{"expire_time", "is_protected", "no_expiry", "source_branch", "source_branch_lsn", "source_branch_time", "ttl"}
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

	// TODO: Enable when PostgresBranchPermission is defined in Task 6
	// if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
	// 	permissions.PostgresBranchName = fmt.Sprintf("${databricks_postgres_branch.%s.name}", key)
	// 	out.Permissions["postgres_branch_"+key] = permissions
	// }

	return nil
}

func init() {
	registerConverter("postgres_branches", postgresBranchConverter{})
}
