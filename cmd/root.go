package cmd

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bricks",
	Short: "Databricks project lifecycle management",
	Long:  `Where's "data"? Secured by the unity catalog. Projects build lifecycle is secured by bricks`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
}

// TODO: replace with zerolog
type levelWriter []string

var logLevel = levelWriter{"[INFO]", "[ERROR]", "[WARN]"}
var verbose bool

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
	if verbose {
		logLevel = append(logLevel, "[DEBUG]")
	}
	ctx := project.Authenticate(context.Background())
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "print debug logs")
	log.SetOutput(&logLevel)
}
