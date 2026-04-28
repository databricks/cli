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

const (
	sourceConfig  = "config"
	sourceEnv     = "env"
	sourceDefault = "default"
)

// TestProcessHook is a test-only seam ProcessUcm calls right before the
// mutator chain runs, with the runtime logdiag context (the one ProcessUcm
// initializes, not the test's pre-init). Production code MUST leave this nil
// — setting it from production is a layering bug. Tests use it to seed
// diagnostics or workspace state that's hard to reproduce via fixtures alone
// (e.g. strict-mode warning behavior in policy-check / validate).
var TestProcessHook func(context.Context, *ucm.Ucm)

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
