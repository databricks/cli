package aircmd

import (
	"time"
)

// buildListRow maps a TrainingWorkflow to a table/JSON row. Optional text
// columns fall back to "-"; host builds the MLflow deep link.
func buildListRow(w *trainingWorkflow, host string) listRow {
	experiment := "-"
	if e := stripExperimentUserPrefix(w.Status.Mlflow.Experiment); e != "" {
		experiment = e
	}

	var startedAt *string
	duration := "-"
	if start := parseRPCTime(w.Status.StartTime); !start.IsZero() {
		s := isoFormat(start)
		startedAt = &s

		// A still-running run has no end_time, so measure against the current time.
		endMillis := time.Now().UnixMilli()
		if end := parseRPCTime(w.Status.EndTime); !end.IsZero() {
			endMillis = end.UnixMilli()
		}
		duration = formatDuration(roundMillisToSeconds(endMillis - start.UnixMilli()))
	}

	mlflowURL := "-"
	if w.Status.Mlflow.ExperimentID != "" && w.Status.Mlflow.RunID != "" {
		mlflowURL = mlflowLogsURL(host, &mlflowIdentifiers{
			ExperimentID: w.Status.Mlflow.ExperimentID,
			RunID:        w.Status.Mlflow.RunID,
		})
	}

	accel := "-"
	if a := acceleratorLabel(w.Spec.Compute.HardwareAcceleratorType, w.Spec.Compute.AcceleratorCount); a != "" {
		accel = a
	}

	return listRow{
		RunID:        w.JobRunID,
		RunName:      w.Status.Job.Name,
		User:         w.Metadata.CreatorName,
		Status:       trainingWorkflowStatus(w.Status.State),
		StartedAt:    startedAt,
		IsSweep:      w.TaskRunID != "",
		Experiment:   experiment,
		Duration:     duration,
		MLflowURL:    mlflowURL,
		Accelerators: accel,
	}
}
