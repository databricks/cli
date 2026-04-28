// Package utils contains helpers shared by `databricks ucm` subcommand
// implementations — primarily ProcessUcm (forked from
// cmd/bundle/utils.ProcessBundle) and ResolveEngineSetting, which picks
// the effective deployment engine.
package utils

import (
	"context"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
)

// PreMutateHook is a deprecated test seam retained as a no-op so the
// existing cmd/ucm/utils/testing_seed_test.go and the verb-test seed
// helpers compile until Task 18 of sub-project A removes them. SETTING
// THIS VARIABLE HAS NO EFFECT ON PRODUCTION CODE PATHS — the new
// ProcessUcm does not call it. Scheduled for removal alongside the test
// migration.
var PreMutateHook func(context.Context, *ucm.Ucm)

const (
	sourceConfig  = "config"
	sourceEnv     = "env"
	sourceDefault = "default"
)

// configureVariables assigns .Value on each named variable via
// Config.InitializeVariables. Errors are surfaced as diagnostics.
func configureVariables(ctx context.Context, u *ucm.Ucm, vars []string) {
	if len(vars) == 0 {
		return
	}
	ucm.ApplyFuncContext(ctx, u, func(ctx context.Context, u *ucm.Ucm) {
		if err := u.Config.InitializeVariables(vars); err != nil {
			logdiag.LogError(ctx, err)
		}
	})
}
