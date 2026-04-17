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
	Status  string `json:"status"` // "pass", "fail", "warn", "info", "skip"
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
		profileName, fromFlag := profileFromCommand(cmd)
		results := runChecks(cmd.Context(), profileName, fromFlag)

		if err := render(cmd.OutOrStdout(), results, root.OutputType(cmd)); err != nil {
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

func render(w io.Writer, results []CheckResult, outputType flags.Output) error {
	switch outputType {
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		buf = append(buf, '\n')
		_, err = w.Write(buf)
		return err
	case flags.OutputText:
		renderText(w, results)
		return nil
	default:
		return fmt.Errorf("unknown output type %s", outputType)
	}
}

func renderText(w io.Writer, results []CheckResult) {
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
		case statusSkip:
			icon = yellow("[skip]")
		}
		msg := fmt.Sprintf("%s %s: %s", icon, bold(r.Name), r.Message)
		if r.Detail != "" {
			msg += fmt.Sprintf(" (%s)", r.Detail)
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
