package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
)

type overrideCompute struct{}

func OverrideCompute() bundle.Mutator {
	return &overrideCompute{}
}

func (m *overrideCompute) Name() string {
	return "OverrideCompute"
}

func overrideJobCompute(j *resources.Job, compute string) {
	for i := range j.Tasks {
		task := &j.Tasks[i]

		if task.ForEachTask != nil {
			task = &task.ForEachTask.Task
		}

		if task.NewCluster != nil || task.ExistingClusterId != "" || task.EnvironmentKey != "" || task.JobClusterKey != "" {
			task.NewCluster = nil
			task.JobClusterKey = ""
			task.EnvironmentKey = ""
			task.ExistingClusterId = compute
		}
	}
}

func (m *overrideCompute) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	if b.Config.Bundle.Mode == config.Production {
		if b.Config.Bundle.ClusterId != "" {
			// Overriding compute via a command-line flag for production works, but is not recommended.
			diags = diags.Extend(diag.Diagnostics{{
				Summary:  "Setting a cluster override for a target that uses 'mode: production' is not recommended",
				Detail:   "It is recommended to always use the same compute for production target for consistency.",
				Severity: diag.Warning,
			}})
		}
	}
	if v := env.Get(ctx, "DATABRICKS_CLUSTER_ID"); v != "" {
		// For historical reasons, we allow setting the cluster ID via the DATABRICKS_CLUSTER_ID
		// when development mode is used. Sometimes, this is done by accident, so we log an info message.
		if b.Config.Bundle.Mode == config.Development {
			cmdio.LogString(ctx, "Setting a cluster override because DATABRICKS_CLUSTER_ID is set. It is recommended to use --cluster-id instead, which works in any target mode.")
		} else {
			// We don't allow using DATABRICKS_CLUSTER_ID in any other mode, it's too error-prone.
			return diag.Warningf("The DATABRICKS_CLUSTER_ID variable is set but is ignored since the current target does not use 'mode: development'")
		}
		b.Config.Bundle.ClusterId = v
	}

	if b.Config.Bundle.ClusterId == "" {
		return diags
	}

	r := b.Config.Resources
	for i := range r.Jobs {
		overrideJobCompute(r.Jobs[i], b.Config.Bundle.ClusterId)
	}

	return diags
}
