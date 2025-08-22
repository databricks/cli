package tfdyn

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func patchApplyPolicyDefaultValues(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
	// If the field "apply_policy_default_values" is not set, do nothing.
	if b, ok := v.Get("apply_policy_default_values").AsBool(); !ok || !b {
		return v, nil
	}

	// If the field "policy_id" is not set, do nothing.
	if _, ok := v.Get("policy_id").AsString(); !ok {
		return v, nil
	}

	// The field "apply_policy_default_values" is set.
	// We need to collect the list of fields that are set explicitly
	// and pass it to Terraform. This enables Terraform to clear
	// server-side defaults from the update request, which in turn
	// allows the backend to re-apply the policy defaults.
	//
	// For more details, see: https://github.com/databricks/terraform-provider-databricks/pull/4834
	//
	paths := dyn.CollectLeafPaths(v)

	// If any of the map or sequence fields are set, always include them entirely instead of traversing the them.
	for _, field := range []string{
		"custom_tags",
		"init_scripts",
		"spark_conf",
		"spark_env_vars",
		"ssh_public_keys",
	} {
		if vv := v.Get(field); vv.IsValid() {
			// Remove all paths that start with the field.
			paths = slices.DeleteFunc(paths, func(p string) bool {
				return strings.HasPrefix(p, field+".") || strings.HasPrefix(p, field+"[")
			})
			// Add the field to the paths.
			paths = append(paths, field)
		}
	}

	sort.Strings(paths)
	valList := make([]dyn.Value, len(paths))
	for i, s := range paths {
		valList[i] = dyn.V(s)
	}
	v, err := dyn.Set(v, "__apply_policy_default_values_allow_list", dyn.V(valList))
	if err != nil {
		return dyn.InvalidValue, err
	}

	return v, nil
}

func convertJobResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the input value to the underlying job schema.
	// This removes superfluous keys and adapts the input to the expected schema.
	vin, diags := convert.Normalize(jobs.JobSettings{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "job normalization diagnostic: %s", diag.Summary)
	}

	// Sort the tasks of each job in the bundle by task key. Sorting
	// the task keys ensures that the diff computed by terraform is correct and avoids
	// recreates. For more details see the NOTE at
	// https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/job#example-usage
	// and https://github.com/databricks/terraform-provider-databricks/issues/4011
	// and https://github.com/databricks/cli/pull/1776
	vout := vin
	var err error
	tasks, ok := vin.Get("tasks").AsSequence()
	if ok {
		sort.Slice(tasks, func(i, j int) bool {
			// We sort the tasks by their task key. Tasks without task keys are ordered
			// before tasks with task keys. We do not error for those tasks
			// since presence of a task_key is validated for in the Jobs backend.
			tk1, ok := tasks[i].Get("task_key").AsString()
			if !ok {
				return true
			}
			tk2, ok := tasks[j].Get("task_key").AsString()
			if !ok {
				return false
			}
			return tk1 < tk2
		})
		vout, err = dyn.Set(vin, "tasks", dyn.V(tasks))
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	// Modify top-level keys.
	vout, err = renameKeys(vout, map[string]string{
		"tasks":        "task",
		"job_clusters": "job_cluster",
		"parameters":   "parameter",
		"environments": "environment",
	})
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Modify keys in the "git_source" block
	vout, err = dyn.Map(vout, "git_source", func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
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
	vout, err = dyn.Map(vout, "task", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Modify "library" blocks for for_each_task
		vout, err = dyn.Map(v, "for_each_task.task", func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
			return renameKeys(v, map[string]string{
				"libraries": "library",
			})
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		return renameKeys(vout, map[string]string{
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

	// Apply __apply_policy_default_values_allow_list for tasks
	vout, err = dyn.Map(vout, "task", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "new_cluster", patchApplyPolicyDefaultValues)
	}))
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Apply __apply_policy_default_values_allow_list for job clusters
	vout, err = dyn.Map(vout, "job_cluster", dyn.Foreach(func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
		return dyn.Map(v, "new_cluster", patchApplyPolicyDefaultValues)
	}))
	if err != nil {
		return dyn.InvalidValue, err
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
