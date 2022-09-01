package fs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/databricks/databricks-sdk-go/workspaces"	
	"github.com/databricks/databricks-sdk-go/service/dbfs"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls <dir-name>",
	Short: "Lists files",
	Long:  `Lists files`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Question (shreyas): Where does the client pick up the login creds from ? Context ?
		// Maybe this client can be added on a higher level ?
		// Create issue to add tests for this ? How to write those for cli ?
		workspacesClient := workspaces.New()
		listStatusResponse, err := workspacesClient.Dbfs.ListStatus(cmd.Context(), 
			dbfs.ListStatusRequest{Path: args[0]},
		)
		if err != nil {
			panic(err)
		}
		files := listStatusResponse.Files
		// TODO: output formatting: JSON, CSV, tables and default
		for _, v := range files {
			fmt.Printf("[-] %s (%d, %v)\n", v.Path, v.FileSize, v.IsDir)
		}
	},
}

func init() {
	// TODO: pietern: conditionally register commands
	// fabianj: don't do it
	fsCmd.AddCommand(lsCmd)
}
