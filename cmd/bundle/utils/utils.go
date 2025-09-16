package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) {
	bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.InitializeVariables(variables)
		if err != nil {
			logdiag.LogError(ctx, err)
		}
	})
}

func InitializeBundle(cmd *cobra.Command) *bundle.Bundle {
	// Load bundle config and apply target
	b := root.MustConfigureBundle(cmd)
	ctx := cmd.Context()
	if logdiag.HasError(ctx) {
		return b
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return b
	}

	// Initialize variables by assigning them values passed as command line flags
	configureVariables(cmd, b, variables)

	engine, err := deploymentEngine(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return b
	}
	b.DirectDeployment = engine == "direct-exp"

	// Set the engine in the user agent
	ctx = useragent.InContext(ctx, "engine", engine)
	cmd.SetContext(ctx)
	return b
}

func deploymentEngine(ctx context.Context) (string, error) {
	engine := os.Getenv("DATABRICKS_CLI_DEPLOYMENT")
	switch engine {
	case "terraform", "":
		return "terraform", nil
	case "direct-exp":
		return "direct-exp", nil
	}
	return "", fmt.Errorf("unexpected setting for DATABRICKS_CLI_DEPLOYMENT=%#v (expected 'terraform' or 'direct-exp' or absent/empty which means 'terraform')", engine)
}
