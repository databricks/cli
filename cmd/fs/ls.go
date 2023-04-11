package fs

import (
	"fmt"

	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls <dir-name>",
	Short: "Lists files",
	Long:  `Lists files`,

	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("TODO")
	},
}

func init() {
	fsCmd.AddCommand(lsCmd)
}
