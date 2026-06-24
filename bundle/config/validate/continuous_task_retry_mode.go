package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

// ContinuousTaskRetryMode warns when a continuous job does not set task_retry_mode.
// task_retry_mode defaults to NEVER, so the tasks of a continuous job are not
// retried on failure unless it is explicitly set to ON_FAILURE.
func ContinuousTaskRetryMode() bundle.ReadOnlyMutator {
	return &continuousTaskRetryMode{}
}

type continuousTaskRetryMode struct{ bundle.RO }

func (v *continuousTaskRetryMode) Name() string {
	return "validate:continuous_task_retry_mode"
}

const continuousTaskRetryWarningSummary = "Continuous job does not set task_retry_mode"

const continuousTaskRetryWarningDetail = `task_retry_mode is not set on this continuous job, so it defaults to NEVER and the job's tasks are not retried on failure.
Set continuous.task_retry_mode to ON_FAILURE to enable task-level retries, or to NEVER to silence this warning.`

func (v *continuousTaskRetryMode) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("continuous"))

	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, continuous dyn.Value) (dyn.Value, error) {
		// An explicit task_retry_mode (NEVER or ON_FAILURE) is a deliberate
		// choice, so we don't warn.
		if continuous.Get("task_retry_mode").IsValid() {
			return continuous, nil
		}

		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Warning,
			Summary:   continuousTaskRetryWarningSummary,
			Detail:    continuousTaskRetryWarningDetail,
			Locations: continuous.Locations(),
			Paths:     []dyn.Path{p},
		})
		return continuous, nil
	})
	if err != nil {
		log.Debugf(ctx, "Error while applying continuous task retry mode validation: %s", err)
	}

	return diags
}
