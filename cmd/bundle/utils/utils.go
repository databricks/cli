package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/spf13/cobra"
)

func GetDefaultVariableFilePath(target string) string {
	return ".databricks/bundle/" + target + "/vars.json"
}

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) diag.Diagnostics {
	return bundle.ApplyFunc(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		err := b.Config.InitializeVariables(variables)
		return diag.FromErr(err)
	})
}

func configureVariablesFromFile(cmd *cobra.Command, b *bundle.Bundle, filePath string) diag.Diagnostics {
	var diags diag.Diagnostics
	return bundle.ApplyFunc(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		f, err := os.ReadFile(filePath)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to read variables file: %w", err))
		}

		val, err := jsonloader.LoadJSON(f, filePath)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to parse variables file: %w", err))
		}

		vars := map[string]any{}
		err = convert.ToTyped(&vars, val)
		if err != nil {
			return diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "failed to parse variables file: " + err.Error(),
				Detail:    "Variables file must be a JSON object with the following format:\n{\"var1\": \"value1\", \"var2\": \"value2\"}",
				Locations: val.Locations(),
			})
		}

		if len(vars) > 0 {
			err = b.Config.InitializeAnyTypeVariables(vars)
			if err != nil {
				return diag.FromErr(err)
			}
		}

		return nil
	})
}

func ConfigureBundleWithVariables(cmd *cobra.Command) (*bundle.Bundle, diag.Diagnostics) {
	// Load bundle config and apply target
	b, diags := root.MustConfigureBundle(cmd)
	if diags.HasError() {
		return b, diags
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		return b, diag.FromErr(err)
	}

	if len(variables) > 0 {
		// Initialize variables by assigning them values passed as command line flags
		diags = diags.Extend(configureVariables(cmd, b, variables))
	}

	variableFilePath, err := cmd.Flags().GetString("var-file")
	if err != nil {
		return b, diag.FromErr(err)
	}

	if variableFilePath == "" {
		// Fallback to default variable file path
		defaultPath := GetDefaultVariableFilePath(b.Config.Bundle.Target)
		normalisedPath := filepath.Join(b.BundleRootPath, defaultPath)
		if _, err := os.Stat(normalisedPath); err == nil {
			variableFilePath = normalisedPath
		}
	}

	if variableFilePath != "" {
		// Initialize variables by loading them from a file
		diags = diags.Extend(configureVariablesFromFile(cmd, b, variableFilePath))
	}

	return b, diags
}
