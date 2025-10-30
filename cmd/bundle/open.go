// Copied to cmd/pipelines/open.go and adapted for pipelines use.
// Consider if changes made here should be made to the pipelines counterpart as well.
package bundle

import (
	"context"
	"errors"
	"fmt"

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
		return "", errors.New("expected a KEY of the resource to open")
	}

	return args[0], nil
}

func newOpenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open a resource in the browser",
		Long: `Open a deployed bundle resource in the Databricks workspace.

Examples:
  databricks bundle open                    # Prompts to select a resource to open
  databricks bundle open my_job             # Open specific job in Workflows UI
  databricks bundle open my_dashboard       # Open dashboard in browser

Use after deployment to quickly navigate to your resources in the workspace.`,
		Args: root.MaximumNArgs(1),
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var arg string
		b, err := utils.ProcessBundle(cmd, &utils.ProcessOptions{
			PostInitFunc: func(ctx context.Context, b *bundle.Bundle) error {
				var err error
				arg, err = resolveOpenArgument(ctx, b, args)
				return err
			},
			InitIDs: true,
		})
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
			return errors.New("resource does not have a URL associated with it (has it been deployed?)")
		}

		cmdio.LogString(cmd.Context(), "Opening browser at "+url)
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
