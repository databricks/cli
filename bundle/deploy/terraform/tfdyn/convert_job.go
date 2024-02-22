package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

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
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Modify keys in the "task" blocks
	vout, err = dyn.Map(vout, "task", dyn.Foreach(func(v dyn.Value) (dyn.Value, error) {
		return renameKeys(v, map[string]string{
			"libraries": "library",
		})
	}))
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Normalize the output value to the target schema.
	vout, diags = convert.Normalize(schema.ResourceJob{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "job normalization diagnostic: %s", diag.Summary)
	}

	return vout, err
}

type jobConverter struct{}

func (jobConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertJobResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Job[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.JobId = fmt.Sprintf("${databricks_job.%s.id}", key)
		out.Permissions["job_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("jobs", jobConverter{})
}
