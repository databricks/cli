package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

const envDeprecationWarning = "Warning: 'databricks auth env' is deprecated and will be removed in a future release.\n"

func newEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "env",
		Short:  "Get authentication environment variables for the current CLI context",
		Hidden: true,
		Long: `Output the environment variables needed to authenticate as the same identity
the CLI is currently authenticated as. This is useful for configuring downstream
tools that accept Databricks authentication via environment variables.

Deprecated: this command will be removed in a future release.`,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(cmd.ErrOrStderr(), envDeprecationWarning)

		_, err := root.MustAnyClient(cmd, args)
		if err != nil {
			return err
		}

		cfg := cmdctx.ConfigUsed(cmd.Context())
		textMode := cmd.Flag("output").Changed && root.OutputType(cmd) == flags.OutputText
		return writeEnvOutput(cmd.OutOrStdout(), auth.Env(cfg), textMode)
	}

	return cmd
}

// writeEnvOutput writes the env var map as sorted KEY=VALUE lines (textMode) or
// indented JSON. In text mode values are quoted for shell safety.
func writeEnvOutput(w io.Writer, envVars map[string]string, textMode bool) error {
	if textMode {
		for _, k := range slices.Sorted(maps.Keys(envVars)) {
			if _, err := fmt.Fprintf(w, "%s=%s\n", k, quoteEnvValue(envVars[k])); err != nil {
				return err
			}
		}
		return nil
	}
	raw, err := json.MarshalIndent(map[string]any{"env": envVars}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(raw))
	return err
}

const shellQuotedSpecialChars = " \t\n\r\"\\$`!#&|;(){}[]<>?*~'"

// quoteEnvValue quotes a value for KEY=VALUE output if it contains spaces or
// shell-special characters. The value is wrapped in single quotes to prevent
// shell expansion; embedded single quotes are escaped POSIX-style by closing
// the quoted string, emitting a backslash-escaped quote, and reopening.
func quoteEnvValue(v string) string {
	if v == "" {
		return `''`
	}
	if !strings.ContainsAny(v, shellQuotedSpecialChars) {
		return v
	}
	return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
}
