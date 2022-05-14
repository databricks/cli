package launch

import (
	"fmt"
	"log"
	"os"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// launchCmd represents the launch command
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launches a notebook on development cluster",
	Long:  `Reads a file and executes it on dev cluster`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contents, err := os.ReadFile(args[0])
		if err != nil {
			log.Fatal(err)
		}
		results := project.RunPythonOnDev(cmd.Context(), string(contents))
		if results.Failed() {
			log.Fatal(results.Error())
		}
		fmt.Println(results.Text())
	},
}

func init() {
	root.RootCmd.AddCommand(launchCmd)
}
