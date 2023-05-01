package flags

import "github.com/spf13/cobra"

const variableFlagDescription = `set values to use for variables defined in bundle config. Example: --var="foo=bar"`

func AddVariableFlag(cmd *cobra.Command, p *[]string) {
	cmd.Flags().StringSliceVar(p, "var", []string{}, variableFlagDescription)
}
