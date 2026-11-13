package aircmd

import (
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// buildListRow extracts the columns shown for one run. Optional cells fall back
// to "-"; MLflowURL starts as "-" and setMLflowLinks fills it in for text output.
// host and workspaceID are used for building dashboard URLs.
func buildListRow(run *jobs.Run, host string, workspaceID int64) listRow {
	experiment := jobExperiment(run)
	experimentName := nilIfEmpty(experiment)
	if experiment == "" {
		experiment = "-"
	}

	startedAtValue := startedAt(run)
	durationSecondsValue := durationSeconds(run)
	duration := "-"
	if durationSecondsValue != nil {
		duration = formatDuration(*durationSecondsValue)
	}

	acceleratorType, numAccelerators := jobCompute(run)
	acceleratorTypeJSON := nilIfEmpty(acceleratorType)
	var numAcceleratorsJSON *int
	if numAccelerators != 0 {
		numAcceleratorsJSON = &numAccelerators
	}
	accel := "-"
	if a := acceleratorLabel(acceleratorType, numAccelerators); a != "" {
		accel = a
	}

	runURL := dashboardURL(host, run.RunId, workspaceID)

	return listRow{
		RunID:           strconv.FormatInt(run.RunId, 10),
		RunName:         run.RunName,
		User:            run.CreatorUserName,
		Status:          displayRunStatus(run),
		DisplayStatus:   displayRunStatus(run),
		StartedAt:       startedAtValue,
		IsSweep:         isSweep(run),
		ExperimentName:  experimentName,
		DurationSeconds: durationSecondsValue,
		AcceleratorType: acceleratorTypeJSON,
		NumAccelerators: numAcceleratorsJSON,
		DashboardURL:    runURL,
		Experiment:      experiment,
		Duration:        duration,
		MLflowURL:       "-",
		MLflowLabel:     "-",
		RunURL:          runURL,
		Accelerators:    accel,
	}
}
