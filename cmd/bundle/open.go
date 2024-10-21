package bundle

import (
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

	"github.com/pkg/browser"
)

func newOpenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open the web UI for a resource",
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

		// If no arguments are specified, prompt the user to select something to run.
		if len(args) == 0 && cmdio.IsPromptSupported(ctx) {
			// Invert completions from KEY -> NAME, to NAME -> KEY.
			inv := make(map[string]string)
			for k, v := range resources.CompletionMap(b) {
				inv[v] = k
			}
			id, err := cmdio.Select(ctx, inv, "Resource to open")
			if err != nil {
				return err
			}
			args = append(args, id)
		}

		if len(args) < 1 {
			return fmt.Errorf("expected a KEY of the resource to open")
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
		resource, err := resources.Lookup(b, args[0])
		if err != nil {
			return err
		}

		// Confirm that the resource has a URL.
		url := resource.GetURL()
		if url == "" {
			return errors.New("resource does not have a URL associated with it (has it been deployed?)")
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
			return resources.Completions(b), cobra.ShellCompDirectiveNoFileComp
		} else {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return cmd
}
