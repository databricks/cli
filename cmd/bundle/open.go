package bundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"github.com/pkg/browser"
)

func promptOpenArgument(ctx context.Context, b *bundle.Bundle) (string, error) {
	// Compute map of "Human readable name of resource" -> "resource key".
	inv := make(map[string]string)
	for k, ref := range resources.Completions(b) {
		title := fmt.Sprintf("%s: %s", ref.Description.SingularTitle, ref.Resource.GetName())
		inv[title] = k
	}

	key, err := cmdio.Select(ctx, inv, "Resource to open")
	if err != nil {
		return "", err
	}

	return key, nil
}

func resolveOpenArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, error) {
	// If no arguments are specified, prompt the user to select the resource to open.
	if len(args) == 0 && cmdio.IsPromptSupported(ctx) {
		return promptOpenArgument(ctx, b)
	}

	if len(args) < 1 {
		return "", fmt.Errorf("expected a KEY of the resource to open")
	}

	return args[0], nil
}

func newOpenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open a resource in the browser",
		Args:  root.MaximumNArgs(1),
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = bundle.Apply(ctx, b, phases.Initialize())
		if err := diags.Error(); err != nil {
			return err
		}

		arg, err := resolveOpenArgument(ctx, b, args)
		if err != nil {
			return err
		}

		cacheDir, err := terraform.Dir(ctx, b)
		if err != nil {
			return err
		}
		_, stateFileErr := os.Stat(filepath.Join(cacheDir, terraform.TerraformStateFileName))
		_, configFileErr := os.Stat(filepath.Join(cacheDir, terraform.TerraformConfigFileName))
		noCache := errors.Is(stateFileErr, os.ErrNotExist) || errors.Is(configFileErr, os.ErrNotExist)

		if forcePull || noCache {
			diags = bundle.Apply(ctx, b, bundle.Seq(
				terraform.StatePull(),
				terraform.Interpolate(),
				terraform.Write(),
			))
			if err := diags.Error(); err != nil {
				return err
			}
		}

		diags = bundle.Apply(ctx, b, bundle.Seq(
			terraform.Load(),
			mutator.InitializeURLs(),
		))
		if err := diags.Error(); err != nil {
			return err
		}

		// Locate resource to open.
		ref, err := resources.Lookup(b, arg)
		if err != nil {
			return err
		}

		// Confirm that the resource has a URL.
		url := ref.Resource.GetURL()
		if url == "" {
			return fmt.Errorf("resource does not have a URL associated with it (has it been deployed?)")
		}

		return browser.OpenURL(url)
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		b, diags := root.MustConfigureBundle(cmd)
		if err := diags.Error(); err != nil {
			cobra.CompErrorln(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		// No completion in the context of a bundle.
		// Source and destination paths are taken from bundle configuration.
		if b == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 0 {
			completions := resources.Completions(b)
			return maps.Keys(completions), cobra.ShellCompDirectiveNoFileComp
		} else {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return cmd
}
