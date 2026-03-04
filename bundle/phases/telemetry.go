package phases

import (
	"context"
	"slices"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonsaver"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

func getExecutionTimes(b *bundle.Bundle) []protos.IntMapEntry {
	executionTimes := b.Metrics.ExecutionTimes

	// Sort the execution times in descending order.
	sort.Slice(executionTimes, func(i, j int) bool {
		return executionTimes[i].Value > executionTimes[j].Value
	})

	// Keep only the top 250 execution times. This keeps the telemetry event
	// reasonable in size. This should be unnecessary in most cases but is
	// done out of caution since the number of mutators depends upon user input.
	// Eg: every pattern in `includes:` triggers a new mutator.
	if len(executionTimes) > 250 {
		executionTimes = executionTimes[:250]
	}

	return executionTimes
}

func logDeployTelemetry(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) {
	resourcesCount := int64(0)
	_, err := dyn.MapByPattern(b.Config.Value(), dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		resourcesCount++
		return v, nil
	})
	if err != nil {
		log.Debugf(ctx, "failed to count resources: %s", err)
	}

	var jobsIds []string
	for _, job := range b.Config.Resources.Jobs {
		if len(jobsIds) >= ResourceIdLimit {
			break
		}

		// Do not include missing IDs in telemetry. We can still detect them
		// by comparing against the resource count.
		if job == nil || job.ID == "" {
			continue
		}
		jobsIds = append(jobsIds, job.ID)
	}
	var pipelineIds []string
	for _, pipeline := range b.Config.Resources.Pipelines {
		if len(pipelineIds) >= ResourceIdLimit {
			break
		}

		// Do not include missing IDs in telemetry. We can still detect them
		// by comparing against the resource count.
		if pipeline == nil || pipeline.ID == "" {
			continue
		}
		pipelineIds = append(pipelineIds, pipeline.ID)
	}
	var clusterIds []string
	for _, cluster := range b.Config.Resources.Clusters {
		if len(clusterIds) >= ResourceIdLimit {
			break
		}

		// Do not include missing IDs in telemetry. We can still detect them
		// by comparing against the resource count.
		if cluster == nil || cluster.ID == "" {
			continue
		}
		clusterIds = append(clusterIds, cluster.ID)
	}
	var dashboardIds []string
	for _, dashboard := range b.Config.Resources.Dashboards {
		if len(dashboardIds) >= ResourceIdLimit {
			break
		}

		// Do not include missing IDs in telemetry. We can still detect them
		// by comparing against the resource count.
		if dashboard == nil || dashboard.ID == "" {
			continue
		}
		dashboardIds = append(dashboardIds, dashboard.ID)
	}

	// sort the IDs to make the record generated deterministic
	// this is important for testing purposes
	slices.Sort(jobsIds)
	slices.Sort(pipelineIds)
	slices.Sort(clusterIds)
	slices.Sort(dashboardIds)

	// If the bundle UUID is not set, we use a default 0 value.
	bundleUuid := "00000000-0000-0000-0000-000000000000"
	if b.Config.Bundle.Uuid != "" {
		bundleUuid = b.Config.Bundle.Uuid
	}

	variableCount := len(b.Config.Variables)
	complexVariableCount := int64(0)
	lookupVariableCount := int64(0)
	for _, v := range b.Config.Variables {
		// If the resolved value of the variable is a complex type, we count it as a complex variable.
		// We can't rely on the "type: complex" annotation because the annotation is optional in some contexts
		// like bundle YAML files.
		if v.IsComplexValued() {
			complexVariableCount++
		}

		if v.Lookup != nil {
			lookupVariableCount++
		}
	}

	artifactPathType := protos.BundleDeployArtifactPathTypeUnspecified
	if libraries.IsVolumesPath(b.Config.Workspace.ArtifactPath) {
		artifactPathType = protos.BundleDeployArtifactPathTypeVolume
	} else if libraries.IsWorkspacePath(b.Config.Workspace.ArtifactPath) {
		artifactPathType = protos.BundleDeployArtifactPathTypeWorkspace
	}

	mode := protos.BundleModeUnspecified
	switch b.Config.Bundle.Mode {
	case config.Development:
		mode = protos.BundleModeDevelopment
	case config.Production:
		mode = protos.BundleModeProduction
	}

	experimentalConfig := b.Config.Experimental
	if experimentalConfig == nil {
		experimentalConfig = &config.Experimental{}
	}

	telemetry.Log(ctx, protos.DatabricksCliLog{
		BundleDeployEvent: &protos.BundleDeployEvent{
			BundleUuid:   bundleUuid,
			DeploymentId: b.Metrics.DeploymentId.String(),

			ResourceCount:                     resourcesCount,
			ResourceJobCount:                  int64(len(b.Config.Resources.Jobs)),
			ResourcePipelineCount:             int64(len(b.Config.Resources.Pipelines)),
			ResourceModelCount:                int64(len(b.Config.Resources.Models)),
			ResourceExperimentCount:           int64(len(b.Config.Resources.Experiments)),
			ResourceModelServingEndpointCount: int64(len(b.Config.Resources.ModelServingEndpoints)),
			ResourceRegisteredModelCount:      int64(len(b.Config.Resources.RegisteredModels)),
			ResourceQualityMonitorCount:       int64(len(b.Config.Resources.QualityMonitors)),
			ResourceSchemaCount:               int64(len(b.Config.Resources.Schemas)),
			ResourceVolumeCount:               int64(len(b.Config.Resources.Volumes)),
			ResourceClusterCount:              int64(len(b.Config.Resources.Clusters)),
			ResourceDashboardCount:            int64(len(b.Config.Resources.Dashboards)),
			ResourceAppCount:                  int64(len(b.Config.Resources.Apps)),

			ResourceJobIDs:       jobsIds,
			ResourcePipelineIDs:  pipelineIds,
			ResourceClusterIDs:   clusterIds,
			ResourceDashboardIDs: dashboardIds,

			Experimental: &protos.BundleDeployExperimental{
				BundleMode:                   mode,
				ConfigurationFileCount:       b.Metrics.ConfigurationFileCount,
				TargetCount:                  b.Metrics.TargetCount,
				WorkspaceArtifactPathType:    artifactPathType,
				BoolValues:                   b.Metrics.BoolValues,
				LocalCacheMeasurementsMs:     b.Metrics.LocalCacheMeasurementsMs,
				PythonAddedResourcesCount:    b.Metrics.PythonAddedResourcesCount,
				PythonUpdatedResourcesCount:  b.Metrics.PythonUpdatedResourcesCount,
				PythonResourceLoadersCount:   int64(len(experimentalConfig.Python.Resources)),
				PythonResourceMutatorsCount:  int64(len(experimentalConfig.Python.Mutators)),
				VariableCount:                int64(variableCount),
				ComplexVariableCount:         complexVariableCount,
				LookupVariableCount:          lookupVariableCount,
				BundleMutatorExecutionTimeMs: getExecutionTimes(b),
				ResourceStateSizeBytes:       computeResourceStateSizes(ctx, b),
				ResourceActionCounts:         computeActionCounts(plan),
			},
		},
	})
}

// computeActionCounts computes per-resource-type action counts from the deploy plan.
func computeActionCounts(plan *deployplan.Plan) []protos.IntMapEntry {
	if plan == nil {
		return nil
	}

	counts := map[string]int64{}
	for key, entry := range plan.Plan {
		if entry.Action == deployplan.Skip {
			continue
		}

		resourceType := config.GetResourceTypeFromKey(key)
		if resourceType == "" {
			continue
		}

		mapKey := resourceType + "." + string(entry.Action)
		counts[mapKey]++
	}

	return sortedIntMapEntries(counts)
}

// computeResourceStateSizes computes per-resource config sizes in bytes from
// the bundle's YAML configuration. Returns one entry per resource instance,
// sorted in ascending order. This is engine-independent since it uses the
// YAML config directly.
func computeResourceStateSizes(ctx context.Context, b *bundle.Bundle) []int64 {
	var sizes []int64

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			jsonBytes, err := jsonsaver.Marshal(v)
			if err != nil {
				log.Debugf(ctx, "failed to marshal resource %s: %s", p, err)
				return v, nil
			}
			sizes = append(sizes, int64(len(jsonBytes)))
			return v, nil
		},
	)
	if err != nil {
		log.Debugf(ctx, "failed to compute resource state sizes: %s", err)
		return nil
	}

	slices.Sort(sizes)
	return sizes
}

// sortedIntMapEntries converts a map to a sorted slice of IntMapEntry.
func sortedIntMapEntries(m map[string]int64) []protos.IntMapEntry {
	entries := make([]protos.IntMapEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, protos.IntMapEntry{Key: k, Value: v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return entries
}
