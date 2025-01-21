package bundle

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/databricks/cli/cmd/bundle/utils"
)

func initVariableFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in bundle config. Example: --var="foo=bar"`)
	cmd.PersistentFlags().String("var-file", "", fmt.Sprintf(`path to a JSON file containing variables. Example: --var-file="/path/to/vars.json" (default "%s")`, utils.GetDefaultVariableFilePath("<target>")))
}
