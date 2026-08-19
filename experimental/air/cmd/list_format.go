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
	experiment := "-"
	if e := jobExperiment(run); e != "" {
		experiment = e
	}

	var startedAt *string
	duration := "-"
	if start, end := jobTiming(run); start > 0 {
		s := isoFormat(time.UnixMilli(start))
		startedAt = &s
		if end == 0 {
			// Still running: measure against the current time.
			end = time.Now().UnixMilli()
		}
		duration = formatDuration(roundMillisToSeconds(end - start))
	}

	accel := "-"
	if a := acceleratorLabel(jobCompute(run)); a != "" {
		accel = a
	}

	return listRow{
		RunID:        strconv.FormatInt(run.RunId, 10),
		RunName:      run.RunName,
		User:         run.CreatorUserName,
		Status:       runStatus(run.State),
		StartedAt:    startedAt,
		IsSweep:      isSweep(run),
		Experiment:   experiment,
		Duration:     duration,
		MLflowURL:    "-",
		MLflowLabel:  "-",
		RunURL:       dashboardURL(host, run.RunId, workspaceID),
		Accelerators: accel,
	}
}
