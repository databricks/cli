// Package utils contains helpers shared by `databricks ucm` subcommand
// implementations — primarily ProcessUcm (forked from
// cmd/bundle/utils.ProcessBundle) and ResolveEngineSetting, which picks
// the effective deployment engine.
package utils

import (
	"context"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/spf13/cobra"
)

// PreMutateHook is a deprecated test-only seam. Pre-fork ProcessUcm called it
// to install fake CurrentUser / WorkspaceClient injectors in unit tests; the
// new ProcessUcm fork (#98) routes those concerns through MustConfigureUcm
// and a context-scoped Ucm seed instead. PreMutateHook is retained as a
// no-op until cmd/ucm tests migrate (tracked in sub-project A.iv, Task 18).
//
// MUST NOT be set outside of tests.
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

// varsFromCmd reads the `--var` StringSlice flag if it is wired on cmd.
// Returns nil if the flag is absent (e.g. tests that build cmds without it).
func varsFromCmd(cmd *cobra.Command) []string {
	f := cmd.Flag("var")
	if f == nil {
		return nil
	}
	vals, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		return nil
	}
	return vals
}

// getTargetFromCmd returns the target name from command flags.
func getTargetFromCmd(cmd *cobra.Command) string {
	if flag := cmd.Flag("target"); flag != nil {
		return flag.Value.String()
	}
	return ""
}
