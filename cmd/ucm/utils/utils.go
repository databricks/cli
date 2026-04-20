// Package utils contains helpers shared by `databricks ucm` subcommand
// implementations — primarily ProcessUcm, which mirrors the role of
// cmd/bundle/utils.ProcessBundle for the ucm verbs.
package utils

import (
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

// ProcessOptions controls optional behavior in ProcessUcm. M0 only exposes
// the `Validate` switch; more knobs will join as additional verbs land.
type ProcessOptions struct {
	// Validate runs phases.Validate after loading the target. Set this when
	// implementing the `validate` or `policy-check` verbs.
	Validate bool
}

// ProcessUcm loads the ucm.yml rooted at the working directory (or
// DATABRICKS_UCM_ROOT), selects the target indicated by --target (or the
// default target), and — if opts.Validate — runs the validation phase.
//
// Errors are reported via logdiag. The caller should check
// logdiag.HasError(cmd.Context()) and render diagnostics before returning.
func ProcessUcm(cmd *cobra.Command, opts ProcessOptions) *ucm.Ucm {
	ctx := cmd.Context()
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}

	u := ucm.MustLoad(ctx)
	if u == nil || logdiag.HasError(ctx) {
		return u
	}

	if target := getTargetFromCmd(cmd); target == "" {
		phases.LoadDefaultTarget(ctx, u)
	} else {
		phases.LoadNamedTarget(ctx, u, target)
	}
	if logdiag.HasError(ctx) {
		return u
	}

	if opts.Validate {
		phases.Validate(ctx, u)
	}
	return u
}

func getTargetFromCmd(cmd *cobra.Command) string {
	if flag := cmd.Flag("target"); flag != nil {
		return flag.Value.String()
	}
	return ""
}
