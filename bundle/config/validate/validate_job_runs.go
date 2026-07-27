package validate

import (
	"context"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

func ValidateJobRuns() bundle.ReadOnlyMutator {
	return &validateJobRuns{}
}

type validateJobRuns struct{ bundle.RO }

func (v *validateJobRuns) Name() string {
	return "validate:job_runs"
}

func (v *validateJobRuns) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	d := pathDiags{b: b}

	for name, jr := range b.Config.Resources.JobRuns {
		// An empty `job_runs.<name>:` entry loads as a present key with a nil value.
		if jr == nil {
			continue
		}
		prefix := "resources.job_runs." + name

		// idempotency_token is computed in DoCreate; rerun_token is the supported
		// way to force a run.
		if jr.IdempotencyToken != "" {
			d.errorf(prefix+".idempotency_token",
				"idempotency_token is computed automatically and must not be set in bundle configuration; set `rerun_token` to force a new run")
		}

		if jr.Timeout != "" {
			if _, err := time.ParseDuration(jr.Timeout); err != nil {
				d.errorf(prefix+".timeout",
					"timeout must be a duration such as \"30m\" or \"2h\": %s", err)
			}
		}
	}

	v.checkStateReferences(b, &d)

	return d.sorted()
}

// checkStateReferences rejects references to the `state` of a run configured
// with wait_for_completion: false, which is still in flight when the dependent
// resources are created.
func (v *validateJobRuns) checkStateReferences(b *bundle.Bundle, d *pathDiags) {
	_ = dyn.WalkReadOnly(b.Config.Value(), func(path dyn.Path, val dyn.Value) error {
		ref, ok := dynvar.NewRef(val)
		if !ok {
			return nil
		}
		for _, r := range ref.References() {
			name, ok := jobRunStateReference(r)
			if !ok {
				continue
			}
			jr := b.Config.Resources.JobRuns[name]
			if jr == nil || jr.WaitForCompletion == nil || *jr.WaitForCompletion {
				continue
			}
			d.errorAt(path, []dyn.Location{val.Location()},
				"%q: the run's state is not known because resources.job_runs.%s sets wait_for_completion: false", r, name)
		}
		return nil
	})
}

// jobRunStateReference returns the run name when ref points into a job run's
// `state`, e.g. "resources.job_runs.nightly.state.result_state".
func jobRunStateReference(ref string) (string, bool) {
	parts := strings.Split(ref, ".")
	if len(parts) < 5 || parts[0] != "resources" || parts[1] != "job_runs" || parts[3] != "state" {
		return "", false
	}
	return parts[2], true
}
