package aircmd

import (
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// listedRunFields is a list entry with the task-filter inputs it was built from.
type listedRunFields struct {
	entry  listedRun
	fields filterFields
}

// buildListRows builds one list entry per AI runtime task of the run, in task
// order, each read from that task's own task runs (see taskRun). A run without
// an AI runtime task yields a single entry read from the run as is.
func buildListRows(run *jobs.Run, host string, workspaceID int64) []listedRunFields {
	keys := airTaskKeys(run)
	if len(keys) == 0 {
		return []listedRunFields{listedRunFor(run, host, workspaceID)}
	}
	entries := make([]listedRunFields, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, listedRunFor(taskRun(run, key), host, workspaceID))
	}
	return entries
}

// listedRunFor builds the list entry for a run (or a run narrowed to one task).
func listedRunFor(run *jobs.Run, host string, workspaceID int64) listedRunFields {
	row := buildListRow(run, host, workspaceID)
	return listedRunFields{
		entry:  listedRun{row: row, runID: run.RunId, taskKey: row.TaskKey, tasks: retryTasks(run), terminal: isTerminal(run)},
		fields: filterFieldsFromRun(run),
	}
}

// buildListRow extracts the columns shown for one run, read from its first AI
// runtime task. Optional cells fall back to "-"; MLflowURL starts as "-" and
// setMLflowLinks fills it in for text output. host and workspaceID are used for
// building dashboard URLs.
func buildListRow(run *jobs.Run, host string, workspaceID int64) listRow {
	taskKey := ""
	if t := airWrapperTask(run); t != nil {
		taskKey = t.TaskKey
	}
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
		TaskKey:         taskKey,
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
