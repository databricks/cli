package ucm

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

func newDriftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Detect out-of-band Unity Catalog changes by comparing persisted state with live UC reads.",
		Long: `Detect out-of-band Unity Catalog changes by comparing persisted state with live UC reads.

For every resource recorded in the direct-engine state file, ucm fetches the
live UC object through the Databricks SDK and reports per-field mismatches.
Use this periodically or from CI to catch manual UI/API edits that did not go
through ucm.

Common invocations:
  databricks ucm drift                   # Check the default target
  databricks ucm drift --target prod     # Check a specific target
  databricks ucm drift -o json           # Emit structured JSON for tooling

Exit codes:
  0  no drift detected
  1  at least one resource has drifted

Note: drift currently operates on direct-engine state only. Terraform-engine
drift requires parsing generic attribute maps from tfstate and is a follow-up;
the command still routes all live reads through the SDK regardless of engine.`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u, err := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if err != nil {
			return err
		}
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve deploy options: %w", err)
		}

		report := phases.Drift(ctx, u, opts)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}
		if report == nil {
			return root.ErrAlreadyPrinted
		}

		out := cmd.OutOrStdout()
		switch root.OutputType(cmd) {
		case flags.OutputJSON:
			if err := renderDriftJSON(out, report); err != nil {
				return err
			}
		default:
			renderDriftText(out, report)
		}

		if report.HasDrift() {
			// Return ErrAlreadyPrinted so root.Execute exits 1 without
			// prepending a second "Error: ..." line on top of our own report.
			return root.ErrAlreadyPrinted
		}
		return nil
	}

	return cmd
}

// renderDriftText emits the human-readable report.
//
//	DRIFT DETECTED on N resource(s):
//
//	resources.catalogs.sales:
//	  comment: state="..." live="..."
//
// With no drift, prints a one-liner so CI logs carry a positive signal.
func renderDriftText(out io.Writer, report *direct.Report) {
	if !report.HasDrift() {
		fmt.Fprintln(out, "No drift detected.")
		return
	}
	fmt.Fprintf(out, "DRIFT DETECTED on %d resource(s):\n\n", len(report.Drift))
	for i, rd := range report.Drift {
		fmt.Fprintf(out, "%s:\n", rd.Key)
		for _, f := range rd.Fields {
			fmt.Fprintf(out, "  %s: state=%s live=%s\n", f.Field, formatDriftValue(f.State), formatDriftValue(f.Live))
		}
		if i < len(report.Drift)-1 {
			fmt.Fprintln(out)
		}
	}
}

// renderDriftJSON emits `{"drift": [...]}` to stdout. MarshalIndent keeps
// the payload human-eyeable when piped through `jq .` or captured in CI logs.
func renderDriftJSON(out io.Writer, report *direct.Report) error {
	buf, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(buf))
	return nil
}

// formatDriftValue renders a drift value for the text path. Strings are
// quoted so `comment: state=""` is unambiguous; string maps render with
// the `{k="v",...}` shape so multi-key drift stays on one line. Other
// types fall through to Go's default formatting.
func formatDriftValue(v any) string {
	switch x := v.(type) {
	case string:
		return fmt.Sprintf("%q", x)
	case map[string]string:
		if len(x) == 0 {
			return "{}"
		}
		parts := make([]string, 0, len(x))
		for k, val := range x {
			parts = append(parts, fmt.Sprintf("%s=%q", k, val))
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		return fmt.Sprintf("%v", v)
	}
}
