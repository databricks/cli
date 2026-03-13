package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CheckResult holds the outcome of a single diagnostic check.
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn", "info"
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// New returns the doctor command.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "doctor",
		Args:          root.NoArgs,
		Short:         "Validate your Databricks CLI setup",
		GroupID:       "development",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		results := runChecks(cmd)

		switch root.OutputType(cmd) {
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			buf = append(buf, '\n')
			_, err = cmd.OutOrStdout().Write(buf)
			if err != nil {
				return err
			}
		case flags.OutputText:
			renderResults(cmd.OutOrStdout(), results)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		if hasFailedChecks(results) {
			return errors.New("one or more checks failed")
		}
		return nil
	}

	return cmd
}

func renderResults(w io.Writer, results []CheckResult) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	for _, r := range results {
		var icon string
		switch r.Status {
		case statusPass:
			icon = green("[ok]")
		case statusFail:
			icon = red("[FAIL]")
		case statusWarn:
			icon = yellow("[warn]")
		case statusInfo:
			icon = cyan("[info]")
		}
		msg := fmt.Sprintf("%s %s: %s", icon, bold(r.Name), r.Message)
		if r.Detail != "" {
			msg += fmt.Sprintf(" (%s)", r.Detail)
		}
		fmt.Fprintln(w, msg)
	}
}

func hasFailedChecks(results []CheckResult) bool {
	for _, result := range results {
		if result.Status == statusFail {
			return true
		}
	}
	return false
}
