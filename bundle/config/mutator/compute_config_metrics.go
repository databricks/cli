package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

// TODO: Unit tests.
type computeConfigMetrics struct{}

func ComputeConfigMetrics() bundle.Mutator {
	return &computeConfigMetrics{}
}

func (m *computeConfigMetrics) Name() string {
	return "ComputeConfigMetrics"
}

func (m *computeConfigMetrics) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	complexVariableCount, lookupVariableCount := 0, 0
	for _, v := range b.Config.Variables {
		if v.Type == "complex" {
			complexVariableCount++
		}
		if v.Lookup != nil {
			lookupVariableCount++
		}
	}

	// variable metrics
	b.DeployEvent.Metrics.VariableCount = int64(len(b.Config.Variables))
	b.DeployEvent.Metrics.ComplexVariableCount = int64(complexVariableCount)
	b.DeployEvent.Metrics.LookupVariableCount = int64(lookupVariableCount)

	// resource metrics
	// TODO: Add automated unit test.
	resourceCount := 0
	_, err := dyn.MapByPattern(b.Config.Value(), dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		resourceCount++
		return v, nil
	})
	if err != nil {
		log.Debugf(ctx, "Error counting resources: %s", err)
	}
	b.DeployEvent.Metrics.ResourceCount = int64(resourceCount)
	b.DeployEvent.Metrics.JobCount = int64(len(b.Config.Resources.Jobs))
	b.DeployEvent.Metrics.PipelineCount = int64(len(b.Config.Resources.Pipelines))
	b.DeployEvent.Metrics.ModelCount = int64(len(b.Config.Resources.Models))
	b.DeployEvent.Metrics.ExperimentCount = int64(len(b.Config.Resources.Experiments))
	b.DeployEvent.Metrics.ModelServingEndpointCount = int64(len(b.Config.Resources.ModelServingEndpoints))
	b.DeployEvent.Metrics.RegisteredModelCount = int64(len(b.Config.Resources.RegisteredModels))
	b.DeployEvent.Metrics.QualityMonitorCount = int64(len(b.Config.Resources.QualityMonitors))
	b.DeployEvent.Metrics.SchemaCount = int64(len(b.Config.Resources.Schemas))
	b.DeployEvent.Metrics.VolumeCount = int64(len(b.Config.Resources.Volumes))
	b.DeployEvent.Metrics.ClusterCount = int64(len(b.Config.Resources.Clusters))
	b.DeployEvent.Metrics.DashboardCount = int64(len(b.Config.Resources.Dashboards))

	// Log usage of experimental features.
	preInitScriptCount, postInitScriptCount, preBuildScriptCount, postBuildScriptCount, preDeployScriptCount, postDeployScriptCount := 0, 0, 0, 0, 0, 0
	for k := range b.Config.Experimental.Scripts {
		switch k {
		case config.ScriptPreInit:
			preInitScriptCount++
		case config.ScriptPostInit:
			postInitScriptCount++
		case config.ScriptPreBuild:
			preBuildScriptCount++
		case config.ScriptPostBuild:
			postBuildScriptCount++
		case config.ScriptPreDeploy:
			preDeployScriptCount++
		case config.ScriptPostDeploy:
			postDeployScriptCount++
		}
	}
	b.DeployEvent.Metrics.PreinitScriptCount = int64(preInitScriptCount)
	b.DeployEvent.Metrics.PostinitScriptCount = int64(postInitScriptCount)
	b.DeployEvent.Metrics.PrebuildScriptCount = int64(preBuildScriptCount)
	b.DeployEvent.Metrics.PostbuildScriptCount = int64(postBuildScriptCount)
	b.DeployEvent.Metrics.PredeployScriptCount = int64(preDeployScriptCount)
	b.DeployEvent.Metrics.PostdeployScriptCount = int64(postDeployScriptCount)

	b.DeployEvent.Configuration.BoolValues.Append("experimental.use_legacy_run_as", b.Config.Experimental.UseLegacyRunAs)
	b.DeployEvent.Configuration.BoolValues.Append("experimental.python_wheel_wrapper", b.Config.Experimental.PythonWheelWrapper)
	b.DeployEvent.Configuration.BoolValues.Append("experimental.pydabs.enabled", b.Config.Experimental.PyDABs.Enabled)
	b.DeployEvent.Configuration.SetFields.Append("environments", len(b.Config.Environments) > 0)

	// Count number of targets in the bundle configuration:
	b.DeployEvent.Metrics.TargetCount = int64(len(b.Config.Targets))

	// Bundle UUID
	b.DeployEvent.Configuration.BundleUuid = b.Config.Bundle.Uuid

	return nil
}
