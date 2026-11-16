package tui

import (
	"fmt"
	"io"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

// profileEnvVars are the variables cmdio and lipgloss consult when deciding
// on color and background, so a mismatch in a live terminal can be traced.
var profileEnvVars = []string{
	"TERM", "COLORTERM", "CLICOLOR", "CLICOLOR_FORCE", "NO_COLOR",
	"CI", "TERM_PROGRAM", "WT_SESSION",
}

func newProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profile",
		Short: "Print the color decision cmdio makes for stdout and stderr",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmd.OutOrStdout()
			for _, s := range []struct {
				name string
				out  io.Writer
			}{
				{"stdout", cmd.OutOrStdout()},
				{"stderr", cmd.ErrOrStderr()},
			} {
				r, color := cmdio.NewRenderer(ctx, s.out)
				fmt.Fprintf(w, "%s: supports_color=%t profile=%s dark_background=%t\n",
					s.name, color, r.ColorProfile().Name(), r.HasDarkBackground())
			}
			fmt.Fprintln(w, "env:")
			for _, name := range profileEnvVars {
				value := env.Get(ctx, name)
				if value == "" {
					value = "(unset or empty)"
				}
				fmt.Fprintf(w, "  %s=%s\n", name, value)
			}
			return nil
		},
	}
}
