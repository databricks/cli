package dlt

import (
	"fmt"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dlt",
		Short: "DLT CLI",
		Long:  "DLT CLI (stub, to be filled in)",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add 'init' stub command (same description as bundle init)
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new DLT project in the current directory",
		Long:  "Initialize a new DLT project in the current directory. This is a stub for future implementation.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("dlt init is not yet implemented. This will initialize a new DLT project in the future.")
		},
	}
	cmd.AddCommand(initCmd)

	return cmd
}
