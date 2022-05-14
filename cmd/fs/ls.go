package fs

import (
	"fmt"

	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"

	"github.com/databrickslabs/terraform-provider-databricks/storage"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Lists files",
	Long:  `Lists files`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		api := storage.NewDbfsAPI(cmd.Context(), project.Current.Client())
		files, err := api.List(args[0], false)
		if err != nil {
			panic(err)
		}
		for _, v := range files {
			fmt.Printf("[-] %s (%d, %v)\n", v.Path, v.FileSize, v.IsDir)
		}
	},
}

func init() {
	fsCmd.AddCommand(lsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
