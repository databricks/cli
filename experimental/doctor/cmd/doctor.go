package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// status identifies a CheckResult's outcome. The string values are part of the
// JSON wire contract emitted by --output json.
type status string

const (
	statusPass status = "pass"
	statusFail status = "fail"
	statusWarn status = "warn"
	statusInfo status = "info"
	statusSkip status = "skip"
)

// CheckResult holds the outcome of a single diagnostic check.
type CheckResult struct {
	Name    string `json:"name,omitempty"`
	Status  status `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Detail  any    `json:"detail,omitempty"`
}

// DoctorReport is the top-level JSON output shape. Wrapping the results in an
// object leaves room to add fields (summary, version, durationMs, ...) without
// breaking callers that already parse the response.
type DoctorReport struct {
	Results []CheckResult `json:"results"`
}

// NewDoctorCmd returns the doctor command.
func NewDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "doctor",
		Args:          root.NoArgs,
		Short:         "Validate your Databricks CLI setup",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		profileName, fromFlag := profileFromCommand(cmd)
		results := runChecks(cmd.Context(), profileName, fromFlag)

		if err := render(cmd.Context(), cmd.OutOrStdout(), results, root.OutputType(cmd)); err != nil {
			return err
		}

		if hasFailedChecks(results) {
			return errors.New("one or more checks failed")
		}
		return nil
	}

	return cmd
}

func profileFromCommand(cmd *cobra.Command) (string, bool) {
	f := cmd.Flag("profile")
	if f == nil || !f.Changed {
		return "", false
	}
	return f.Value.String(), true
}

func render(ctx context.Context, w io.Writer, results []CheckResult, outputType flags.Output) error {
	switch outputType {
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(DoctorReport{Results: results}, "", "  ")
		if err != nil {
			return err
		}
		buf = append(buf, '\n')
		_, err = w.Write(buf)
		return err
	case flags.OutputText:
		renderText(ctx, w, results)
		return nil
	default:
		return fmt.Errorf("unknown output type %s", outputType)
	}
}

func renderText(ctx context.Context, w io.Writer, results []CheckResult) {
	for _, r := range results {
		var icon string
		switch r.Status {
		case statusPass:
			icon = cmdio.Green(ctx, "[ok]")
		case statusFail:
			icon = cmdio.Red(ctx, "[FAIL]")
		case statusWarn:
			icon = cmdio.Yellow(ctx, "[warn]")
		case statusInfo:
			icon = cmdio.Cyan(ctx, "[info]")
		case statusSkip:
			icon = cmdio.Yellow(ctx, "[skip]")
		}
		msg := fmt.Sprintf("%s %s: %s", icon, cmdio.Bold(ctx, r.Name), r.Message)
		if r.Detail != nil {
			msg += fmt.Sprintf(" (%v)", r.Detail)
		}
		fmt.Fprintln(w, msg)
	}
}

func hasFailedChecks(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == statusFail {
			return true
		}
	}
	return false
}
