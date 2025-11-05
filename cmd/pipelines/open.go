// Copied from cmd/bundle/open.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"

	"github.com/databricks/cli/bundle/resources"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"github.com/pkg/browser"
)

// When no arguments are specified, auto-selects a pipeline if there's exactly one,
// otherwise prompts the user to select a pipeline to open.
func resolveOpenArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	if key := autoSelectSinglePipeline(b); key != "" {
		return key, nil
	}

	if cmdio.IsPromptSupported(ctx) {
		return promptResource(ctx, b)
	}
	return "", errors.New("expected a KEY of the pipeline to open")
}

func openCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open [flags] [KEY]",
		Short: "Open a pipeline in the browser",
		Long: `Open a pipeline in the browser, identified by KEY.
KEY is the unique name of the pipeline to open, as defined in its YAML file.
If there is only one pipeline in the project, KEY is optional and the pipeline will be auto-selected.`,
		Args: root.MaximumNArgs(1),
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			AlwaysPull: forcePull,
			InitIDs:    true,
		})
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		arg, err := resolveOpenArgument(ctx, b, args)
		if err != nil {
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
			return errors.New("pipeline does not have a URL associated with it (has it been deployed?)")
		}

		cmdio.LogString(ctx, "Opening browser at "+url)
		return browser.OpenURL(url)
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if logdiag.HasError(cmd.Context()) {
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
