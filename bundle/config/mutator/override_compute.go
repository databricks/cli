package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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
		var task = &j.Tasks[i]

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
			diags = diags.Extend(diag.Warningf("overriding compute for a target that uses 'mode: production' is not recommended"))
		}
		if env.Get(ctx, "DATABRICKS_CLUSTER_ID") != "" {
			// The DATABRICKS_CLUSTER_ID may be set by accident when doing a production deploy.
			// Overriding the cluster in production is almost always a mistake since customers
			// want consistency in production and not compute that is different each deploy.
			// For this reason we log a warning and ignore the environment variable.
			return diag.Warningf("the DATABRICKS_CLUSTER_ID variable is set but is ignored since the current target uses 'mode: production'")
		}
	}
	if v := env.Get(ctx, "DATABRICKS_CLUSTER_ID"); v != "" {
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
