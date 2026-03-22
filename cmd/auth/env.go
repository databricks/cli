package auth

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Get authentication environment variables for the current CLI context",
		Long: `Output the environment variables needed to authenticate as the same identity
the CLI is currently authenticated as. This is useful for configuring downstream
tools that accept Databricks authentication via environment variables.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, err := root.MustAnyClient(cmd, args)
		if err != nil {
			return err
		}

		cfg := cmdctx.ConfigUsed(cmd.Context())
		envVars := auth.Env(cfg)

		// Output KEY=VALUE lines when the user explicitly passes --output text.
		if cmd.Flag("output").Changed && root.OutputType(cmd) == flags.OutputText {
			w := cmd.OutOrStdout()
			keys := slices.Sorted(maps.Keys(envVars))
			for _, k := range keys {
				fmt.Fprintf(w, "%s=%s\n", k, quoteEnvValue(envVars[k]))
			}
			return nil
		}

		raw, err := json.MarshalIndent(envVars, "", "  ")
		if err != nil {
			return err
		}
		_, _ = cmd.OutOrStdout().Write(raw)
		return nil
	}

	return cmd
}

const shellQuotedSpecialChars = " \t\n\r\"\\$`!#&|;(){}[]<>?*~'"

// quoteEnvValue quotes a value for KEY=VALUE output if it contains spaces or
// shell-special characters. Single quotes prevent shell expansion, and
// embedded single quotes use the POSIX-compatible '\" sequence.
func quoteEnvValue(v string) string {
	if v == "" {
		return `''`
	}
	needsQuoting := strings.ContainsAny(v, shellQuotedSpecialChars)
	if !needsQuoting {
		return v
	}
	return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
}
