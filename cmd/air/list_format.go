package aircmd

import (
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// taskEntry is one task's list entry with the task-filter inputs it was built
// from, before any filter is applied.
type taskEntry struct {
	entry  listedRun
	fields filterFields
}

// runGroup is one run's list entries that pass the filters: one per AI runtime
// task, in task order (a run without one has a single entry). Every list path
// passes runs along as groups, so --limit counts runs, sorting keeps a run's
// rows together, and cancel and the cache act on the whole run.
type runGroup struct {
	runID   int64
	entries []listedRun
}

// buildTaskEntries builds one entry per AI runtime task of the run, in task
// order, each read from that task's own task runs (see taskRun). A run without
// an AI runtime task yields a single entry read from the run as is.
func buildTaskEntries(run *jobs.Run, host string, workspaceID int64) []taskEntry {
	keys := airTaskKeys(run)
	if len(keys) == 0 {
		return []taskEntry{taskEntryFor(run, host, workspaceID)}
	}
	entries := make([]taskEntry, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, taskEntryFor(taskRun(run, key), host, workspaceID))
	}
	return entries
}

// taskEntryFor builds the entry for a run narrowed to one task (or a run
// without an AI runtime task).
func taskEntryFor(run *jobs.Run, host string, workspaceID int64) taskEntry {
	row := buildListRow(run, host, workspaceID)
	return taskEntry{
		entry:  listedRun{row: row, runID: run.RunId, taskKey: airTaskKey(run), tasks: retryTasks(run), terminal: isTerminal(run)},
		fields: filterFieldsFromRun(run),
	}
}

// matchingGroup returns the run's entries that pass f, each filtered on its own
// task, or ok false when none do. When the run has several AI runtime tasks,
// every row is marked to show its task, even if the filter leaves one.
func matchingGroup(runID int64, all []taskEntry, f listFilters) (runGroup, bool) {
	g := runGroup{runID: runID}
	for _, e := range all {
		if f.matchesFields(e.fields) {
			entry := e.entry
			entry.row.ShowTask = len(all) > 1
			g.entries = append(g.entries, entry)
		}
	}
	return g, len(g.entries) > 0
}

// rows returns the group's display rows.
func (g runGroup) rows() []listRow {
	rows := make([]listRow, len(g.entries))
	for i, e := range g.entries {
		rows[i] = e.row
	}
	return rows
}

// buildListRow extracts the columns shown for one run, read from its first AI
// runtime task. Optional cells fall back to "-"; MLflowURL starts as "-" and
// setMLflowLinks fills it in for text output. host and workspaceID are used for
// building dashboard URLs.
func buildListRow(run *jobs.Run, host string, workspaceID int64) listRow {
	taskKey := airTaskKey(run)
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
		TaskKey:         nilIfEmpty(taskKey),
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
