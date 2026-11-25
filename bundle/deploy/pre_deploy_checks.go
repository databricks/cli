package deploy

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

// preDeployChecks holds, per resource type, a function that creates the check that
// "bundle plan" and "bundle deploy" run for that type before the deployment state is
// pulled. isPlan is true for "bundle plan", where a check may report a finding that
// blocks a deploy as a warning instead. A check must not modify the configuration,
// because "bundle migrate" does not run it. Checks run in the canonical order of their
// types (the field order of [config.Resources]).
var preDeployChecks = config.NewResourceTypeRegistry[func(isPlan bool) bundle.Mutator]("pre-deploy check")

type preDeployResourceChecks struct {
	isPlan bool
}

// PreDeployResourceChecks returns a mutator that applies the registered pre-deploy
// checks in sequence. Like [bundle.ApplySeqContext], it stops at the first error.
func PreDeployResourceChecks(isPlan bool) bundle.Mutator {
	return &preDeployResourceChecks{isPlan: isPlan}
}

func (m *preDeployResourceChecks) Name() string {
	return "PreDeployResourceChecks"
}

func (m *preDeployResourceChecks) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var checks []bundle.Mutator
	for _, newCheck := range preDeployChecks.All() {
		checks = append(checks, newCheck(m.isPlan))
	}
	bundle.ApplySeqContext(ctx, b, checks...)
	return nil
}
