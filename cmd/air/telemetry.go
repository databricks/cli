package aircmd

import (
	"context"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// logRunEvent records the submission outcome independently of a later --watch
// outcome. cfg has already passed local validation before submission starts.
func logRunEvent(ctx context.Context, cfg *runConfig, snap snapshotResult, runID int64, elapsed time.Duration, err error) {
	perNode, _ := gpusPerNode(gpuType(cfg.Compute.AcceleratorType))
	_, hasDependencies := cfg.inlineDependencies()
	event := &protos.AirRunEvent{
		GPUType:                       cfg.Compute.AcceleratorType,
		NumGPUs:                       cfg.Compute.NumAccelerators,
		NumNodes:                      cfg.Compute.NumAccelerators / perNode,
		HasDockerImage:                cfg.unityCatalogImagePath() != "" || len(cfg.Containers) > 0, // every container sets its own image
		HasCodeSnapshot:               cfg.CodeSource != nil && cfg.CodeSource.Snapshot != nil,
		HasRequirements:               hasDependencies,
		HasParameters:                 len(cfg.Parameters) > 0,
		MaxRetries:                    cfg.maxRetries(),
		HasTimeout:                    cfg.TimeoutMinutes != nil,
		SubmittedSuccessfully:         err == nil,
		SubmitLatencyMs:               elapsed.Milliseconds(),
		CodeSourceUsesGit:             snap.UsesGit,
		CodeSourceSizeBytes:           snap.SizeBytes,
		CodeSourcePackagingDurationMs: snap.PackagingDurationMs,
		CodeSourceUploadDurationMs:    snap.UploadDurationMs,
	}
	if snap.PackagingMode != nil {
		switch *snap.PackagingMode {
		case modeGitArchive:
			event.CodeSourcePackagingMode = protos.AirPackagingModeGitArchive
		case modePlainTar:
			event.CodeSourcePackagingMode = protos.AirPackagingModePlainTar
		}
	}
	if err == nil {
		event.JobRunID = strconv.FormatInt(runID, 10)
	}
	telemetry.Log(ctx, protos.DatabricksCliLog{AirRunEvent: event})
}
