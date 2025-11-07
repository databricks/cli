package debug

import (
	"strings"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func NewStatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "states",
		Short: "Show available state files",
		Args:  root.NoArgs,
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			// when using filers must run Initialize() phase otherwise some paths are not present and this happens:
			// Error: reading terraform.tfstate: opening: relative path escapes root: terraform.tfstate
			SkipInitialize: !forcePull,
			ReadState:      true,
			AlwaysPull:     forcePull,
		}

		_, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		ctx := cmd.Context()

		if err != nil {
			return err
		}

		var stateStrs []string
		for _, state := range stateDesc.AllStates {
			stateStrs = append(stateStrs, state.String())
		}
		cmdio.LogString(ctx, strings.Join(stateStrs, "\n"))
		return nil
	}

	return cmd
}
