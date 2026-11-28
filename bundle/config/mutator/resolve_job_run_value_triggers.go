package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
)

type resolveJobRunValueTriggers struct{}

// ResolveJobRunValueTriggers copies each job_run's interpolated on_value_change values,
// in trigger order, into lifecycle.triggers_state, where the planner compares them by
// position with the values recorded when the run last fired.
func ResolveJobRunValueTriggers() bundle.Mutator {
	return &resolveJobRunValueTriggers{}
}

func (*resolveJobRunValueTriggers) Name() string {
	return "ResolveJobRunValueTriggers"
}

func (*resolveJobRunValueTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, jr := range b.Config.Resources.JobRuns {
		if jr == nil || jr.Lifecycle == nil {
			continue
		}

		var values []string
		for _, trigger := range jr.Lifecycle.Triggers {
			if trigger.OnValueChange != nil {
				values = append(values, *trigger.OnValueChange)
			}
		}
		if values == nil {
			continue
		}

		if jr.Lifecycle.TriggersState == nil {
			jr.Lifecycle.TriggersState = &resources.JobRunTriggersState{}
		}
		jr.Lifecycle.TriggersState.OnValueChange = values
	}
	return nil
}
