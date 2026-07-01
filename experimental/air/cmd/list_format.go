package aircmd

import (
	"strconv"
	"time"
)

// buildListRow extracts the columns shown for one run. Optional text columns fall
// back to "-" so the table stays aligned. MLflow links aren't carried by
// runs/list, so the column shows "-".
func buildListRow(run *jobRun) listRow {
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
		RunID:        strconv.FormatInt(run.RunID, 10),
		RunName:      run.RunName,
		User:         run.CreatorUserName,
		Status:       statusWord(run.State.LifeCycleState, run.State.ResultState),
		StartedAt:    startedAt,
		IsSweep:      isSweep(run),
		Experiment:   experiment,
		Duration:     duration,
		MLflowURL:    "-",
		Accelerators: accel,
	}
}
