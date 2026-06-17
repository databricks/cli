package phases

import (
	"cmp"
	"context"
	"math"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

func getExecutionTimes(b *bundle.Bundle) []protos.IntMapEntry {
	executionTimes := b.Metrics.ExecutionTimes

	// Sort the execution times in descending order.
	slices.SortFunc(executionTimes, func(a, b protos.IntMapEntry) int {
		return cmp.Compare(b.Value, a.Value)
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

// Exclusive upper bounds in bytes for the upload file-size histogram: entry i
// counts files in [bounds[i-1], bounds[i]). The final bound is math.MaxInt64 so
// every file maps to a bucket and the last entry is the >= 64 MiB tail.
//
// A power-of-2 ladder (each bound is 2x the previous) is used because file sizes
// span many orders of magnitude; constant-ratio buckets keep useful resolution
// across the whole range.
//
// The histogram is positional on the wire, so changing this list changes the
// meaning of every entry. Treat the bounds as frozen once the metric has
// adoption, and keep them in sync with the doc on
// BundleDeployExperimental.UploadFileSizes.
var uploadFileSizeBucketBounds = []int64{
	256,           // 256 B
	512,           // 512 B
	1 << 10,       // 1 KiB
	2 << 10,       // 2 KiB
	4 << 10,       // 4 KiB
	8 << 10,       // 8 KiB
	16 << 10,      // 16 KiB
	32 << 10,      // 32 KiB
	64 << 10,      // 64 KiB
	128 << 10,     // 128 KiB
	256 << 10,     // 256 KiB
	512 << 10,     // 512 KiB
	1 << 20,       // 1 MiB
	2 << 20,       // 2 MiB
	4 << 20,       // 4 MiB
	8 << 20,       // 8 MiB
	16 << 20,      // 16 MiB
	32 << 20,      // 32 MiB
	64 << 20,      // 64 MiB
	math.MaxInt64, // catch-all upper bound
}

// uploadFileSizeBucket returns the histogram bucket index for a file of the
// given size in bytes. Bounds are exclusive upper bounds, so a size that equals
// a bound belongs in the next bucket.
func uploadFileSizeBucket(size int64) int {
	i, found := slices.BinarySearch(uploadFileSizeBucketBounds, size)
	if found {
		i++
	}
	return i
}

// sizer is the subset of fileset.File that uploadFileSizeHistogram needs. Taking
// an interface keeps the histogram unit-testable without constructing real files.
type sizer interface {
	Size() (int64, bool)
}

// uploadFileSizeHistogram counts the uploaded files into uploadFileSizeBucketBounds.
// If any file's size cannot be determined the whole histogram is omitted (returns
// nil) rather than emitting a partial, misleading distribution; upload_file_count
// is reported independently. Also returns nil when no files were uploaded (e.g.
// source-linked deployments).
func uploadFileSizeHistogram(files []sizer) []int64 {
	if len(files) == 0 {
		return nil
	}
	hist := make([]int64, len(uploadFileSizeBucketBounds))
	for _, f := range files {
		size, ok := f.Size()
		if !ok {
			return nil
		}
		hist[uploadFileSizeBucket(size)]++
	}
	return hist
}

// LogDeployTelemetry logs a telemetry event for a bundle deploy command.
func LogDeployTelemetry(ctx context.Context, b *bundle.Bundle, errMsg string) {
	errMsg = telemetry.ScrubErrorMessage(errMsg)

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

	for _, app := range b.Config.Resources.Apps {
		if app != nil && app.Lifecycle != nil && app.Lifecycle.Started != nil {
			b.Metrics.SetBoolValue(metrics.AppLifecycleStarted, *app.Lifecycle.Started)
			break
		}
	}

	for _, cluster := range b.Config.Resources.Clusters {
		if cluster != nil && cluster.Lifecycle != nil && cluster.Lifecycle.Started != nil {
			b.Metrics.SetBoolValue(metrics.ClusterLifecycleStarted, *cluster.Lifecycle.Started)
			break
		}
	}

	for _, warehouse := range b.Config.Resources.SqlWarehouses {
		if warehouse != nil && warehouse.Lifecycle != nil && warehouse.Lifecycle.Started != nil {
			b.Metrics.SetBoolValue(metrics.SqlWarehouseLifecycleStarted, *warehouse.Lifecycle.Started)
			break
		}
	}

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

	uploadFileSizers := make([]sizer, len(b.Files))
	for i, f := range b.Files {
		uploadFileSizers[i] = f
	}

	telemetry.Log(ctx, protos.DatabricksCliLog{
		BundleDeployEvent: &protos.BundleDeployEvent{
			BundleUuid:   bundleUuid,
			DeploymentId: b.Metrics.DeploymentId.String(),
			ErrorMessage: errMsg,

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

			ResourcesMetadata: collectResourcesMetadata(ctx, b),

			Experimental: &protos.BundleDeployExperimental{
				BundleMode:                   mode,
				ConfigurationFileCount:       b.Metrics.ConfigurationFileCount,
				UploadFileCount:              int64(len(b.Files)),
				UploadFileSizes:              uploadFileSizeHistogram(uploadFileSizers),
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
			},
		},
	})
}
