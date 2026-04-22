package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/direct"
)

// Drift fetches the direct-engine state for the current target and compares
// it field-by-field against live Unity Catalog reads. The resulting
// *direct.Report is returned for the caller to render. Errors are reported
// via logdiag; on error Drift returns nil.
//
// Drift always routes through the direct SDK client regardless of the
// configured engine — reading live UC state is an SDK-level operation and
// the terraform engine has no native concept of it. The terraform-engine
// path still needs a reader for its own recorded state; until that lands,
// drift on a terraform-engine target reports nothing (the direct state file
// is absent) and the verb's long help calls out the limitation.
func Drift(ctx context.Context, u *ucm.Ucm, opts Options) *direct.Report {
	log.Info(ctx, "Phase: drift")

	factory := opts.directClientFactoryOrDefault()
	client, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("resolve direct client: %w", err))
		return nil
	}

	state, err := direct.LoadState(direct.StatePath(u))
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("load direct state: %w", err))
		return nil
	}

	report, err := direct.ComputeDrift(ctx, client, state)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("compute drift: %w", err))
		return nil
	}
	return report
}
