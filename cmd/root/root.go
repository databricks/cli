package root

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "bricks",
	Short: "Databricks project lifecycle management",
	Long:  `Where's "data"? Secured by the unity catalog. Projects build lifecycle is secured by bricks`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if Verbose {
			logLevel = append(logLevel, "[DEBUG]")
		}
		log.SetOutput(&logLevel)
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
}

// TODO: replace with zerolog
type levelWriter []string

var logLevel = levelWriter{"[INFO]", "[ERROR]", "[WARN]"}

// Verbose means additional debug information, like API logs
var Verbose bool

func (lw *levelWriter) Write(p []byte) (n int, err error) {
	a := string(p)
	for _, l := range *lw {
		if strings.Contains(a, l) {
			return os.Stdout.Write(p)
		}
	}
	return
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// TODO: deferred panic recovery
	ctx := context.Background()
	err := RootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// flags available for every child command
	RootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "print debug logs")
}
