package experimental

import (
	"errors"

	"github.com/spf13/cobra"
)

func newAirCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "air",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New(`the AIR commands have moved out of experimental; use "databricks air" instead`)
		},
	}
}
