package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
)

type markSelectUsed struct{}

// MarkSelectUsed records in telemetry whether the --select flag was used. It runs
// during initialize so the metric is set even for a deploy that loads a prebuilt
// plan and therefore skips RunPlan. Selectors are validated once state is open (an
// early check in process.go) and resolved against the computed plan in
// Plan.FilterToSelected, so resources removed from config (planned for deletion)
// can be selected too.
func MarkSelectUsed() bundle.Mutator {
	return &markSelectUsed{}
}

func (m *markSelectUsed) Name() string {
	return "MarkSelectUsed"
}

func (m *markSelectUsed) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if len(b.Select) > 0 {
		b.Metrics.SetBoolValue(metrics.SelectUsed, true)
	}
	return nil
}
