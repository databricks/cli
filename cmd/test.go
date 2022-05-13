package cmd

import (
	"log"

	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "run tests for the project",
	Long:  `This is longer description of the command`,
	Run: func(cmd *cobra.Command, args []string) {
		
		results := project.RunPythonOnDev(cmd.Context(), `print("hello, world!")`)
		if results.Failed() {
			log.Fatal(results.Error())
		}
		log.Printf("Success: %s", results.Text())
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// testCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// testCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
