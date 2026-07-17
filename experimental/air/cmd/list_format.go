package aircmd

import (
	"strconv"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// buildListRow extracts the columns shown for one run. Optional cells fall back
// to "-".
func buildListRow(run *jobs.Run) listRow {
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
		Accelerators: accel,
	}
}
