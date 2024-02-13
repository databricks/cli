package terraform

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func renameKeys(v dyn.Value, rename map[string]string) (dyn.Value, error) {
	var err error
	var acc = dyn.V(map[string]dyn.Value{})

	nv, err := dyn.Walk(v, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if len(p) == 0 {
			return v, nil
		}

		// Check if this key should be renamed.
		for oldKey, newKey := range rename {
			if p[0] != dyn.Key(oldKey) {
				continue
			}

			// Add the new key to the accumulator.
			p[0] = dyn.Key(newKey)
			acc, err = dyn.SetByPath(acc, p, v)
			if err != nil {
				return dyn.NilValue, err
			}
			return dyn.InvalidValue, dyn.ErrDrop
		}

		// Pass through all other values.
		return v, dyn.ErrSkip
	})

	if err != nil {
		return dyn.InvalidValue, err
	}

	// Merge the accumulator with the original value.
	return merge.Merge(nv, acc)
}

func convertJobResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the input value to the underlying job schema.
	// This removes superfluous keys and adapts the input to the expected schema.
	vin, diags := convert.Normalize(jobs.JobSettings{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "job normalization diagnostic: %s", diag.Summary)
	}

	// Modify top-level keys.
	vout, err := renameKeys(vin, map[string]string{
		"tasks":        "task",
		"job_clusters": "job_cluster",
		"parameters":   "parameter",
	})
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Modify keys in the "git_source" block
	vout, err = dyn.Map(vout, "git_source", func(v dyn.Value) (dyn.Value, error) {
		return renameKeys(v, map[string]string{
			"git_branch":   "branch",
			"git_commit":   "commit",
			"git_provider": "provider",
			"git_tag":      "tag",
			"git_url":      "url",
		})
	})

	// Normalize the output value to the target schema.
	vout, diags = convert.Normalize(schema.ResourceJob{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "job normalization diagnostic: %s", diag.Summary)
	}

	return vout, err
}

func convertPipelineResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Modify top-level keys.
	return renameKeys(vin, map[string]string{
		"libraries":     "library",
		"clusters":      "cluster",
		"notifications": "notification",
	})
}

func convertModelResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	return vin, nil
}

func convertModelServingEndpointResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	return vin, nil
}

func convertRegisteredModelResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	return vin, nil
}
