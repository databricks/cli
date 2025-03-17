package main

import (
	"context"
	"os"
	"strings"

	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func main() {
	ctx := context.Background()
	bundleCmd := bundle.New(clis.General)

	// HACK: copy functionionality from root command
	rootCmd := cmd.New(ctx)
	root.InitTargetFlag(bundleCmd)
	bundleCmd.PersistentPreRunE = rootCmd.PersistentPreRunE

	// HACK: Replace "databricks bundle" with "dab" in all command descriptions
	replaceCommandDescriptions(bundleCmd)

	err := root.Execute(ctx, bundleCmd)
	if err != nil {
		os.Exit(1)
	}
}

// replaceCommandDescriptions recursively replaces "databricks bundle" with "dab" in all command Long descriptions
func replaceCommandDescriptions(cmd *cobra.Command) {
	if cmd.Long != "" {
		cmd.Long = strings.ReplaceAll(cmd.Long, "databricks bundle", "dab")
	}

	// Recursively process all subcommands
	for _, subCmd := range cmd.Commands() {
		replaceCommandDescriptions(subCmd)
	}
}
