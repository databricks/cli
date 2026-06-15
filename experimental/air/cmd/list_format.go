package aircmd

import (
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// listTemplate is the text-mode table for `air list`. It reads from the JSON
// envelope, so every row is reached through ".Data.Rows". cmdio renders this
// through a tabwriter, so the tab-separated columns align. A still-running run
// has no Started time, so that one cell falls back to "-" inline; the other
// optional cells are filled with "-" when building the row.
const listTemplate = `Run ID	Experiment	Status	Started	Duration	MLflow	User	Accelerators
{{range .Data.Rows}}{{.RunID}}	{{.Experiment}}	{{.Status}}	{{if .StartedAt}}{{.StartedAt}}{{else}}-{{end}}	{{.Duration}}	{{.MLflowURL}}	{{.User}}	{{.Accelerators}}
{{end}}`

// buildListRow extracts the columns shown for one run, reusing the shared format
// helpers. Optional text columns fall back to "-" so the table stays aligned;
// MLflowURL starts as "-" and is filled in later for text output.
func buildListRow(run *jobs.Run) listRow {
	experiment := "-"
	if e := experimentName(run); e != nil {
		experiment = *e
	}

	duration := "-"
	if d := durationSeconds(run); d != nil {
		duration = formatDuration(*d)
	}

	accel := accelerators(run)
	if accel == "" {
		accel = "-"
	}

	return listRow{
		RunID:        strconv.FormatInt(run.RunId, 10),
		RunName:      run.RunName,
		User:         run.CreatorUserName,
		Status:       runStatus(run.State),
		StartedAt:    startedAt(run),
		IsSweep:      tasksAreSweep(run.Tasks),
		Experiment:   experiment,
		Duration:     duration,
		MLflowURL:    "-",
		Accelerators: accel,
	}
}
