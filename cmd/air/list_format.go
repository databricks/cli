package aircmd

import (
	"strconv"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// buildListRow extracts the columns shown for one run. Optional cells fall back
// to "-"; MLflowURL starts as "-" and setMLflowLinks fills it in for text output.
// host and workspaceID are used for building dashboard URLs.
func buildListRow(run *jobs.Run, host string, workspaceID int64) listRow {
	var experimentName *string
	experiment := "-"
	if e := jobExperiment(run); e != "" {
		experimentName = &e
		experiment = e
	}

	var startedAt *string
	var durationSeconds *int64
	duration := "-"
	if start, end := jobTiming(run); start > 0 {
		s := isoFormat(time.UnixMilli(start))
		startedAt = &s
		if end == 0 {
			// Still running: measure against the current time.
			end = time.Now().UnixMilli()
		}
		seconds := roundMillisToSeconds(end - start)
		durationSeconds = &seconds
		duration = formatDuration(seconds)
	}

	acceleratorType, acceleratorCount := jobCompute(run)
	var acceleratorTypeJSON *string
	if acceleratorType != "" {
		acceleratorTypeJSON = &acceleratorType
	}
	var acceleratorCountJSON *int
	if acceleratorCount != 0 {
		acceleratorCountJSON = &acceleratorCount
	}
	accel := "-"
	if a := acceleratorLabel(acceleratorType, acceleratorCount); a != "" {
		accel = a
	}

	return listRow{
		RunID:            strconv.FormatInt(run.RunId, 10),
		RunName:          run.RunName,
		User:             run.CreatorUserName,
		Status:           displayRunStatus(run),
		DisplayStatus:    displayRunStatus(run),
		StartedAt:        startedAt,
		IsSweep:          isSweep(run),
		ExperimentName:   experimentName,
		DurationSeconds:  durationSeconds,
		AcceleratorType:  acceleratorTypeJSON,
		AcceleratorCount: acceleratorCountJSON,
		Experiment:       experiment,
		Duration:         duration,
		MLflowURL:        "-",
		MLflowLabel:      "-",
		RunURL:           dashboardURL(host, run.RunId, workspaceID),
		Accelerators:     accel,
	}
}
