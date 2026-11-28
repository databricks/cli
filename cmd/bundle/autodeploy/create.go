package autodeploy

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/gitwebhooks"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/spf13/cobra"
)

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Deploy this bundle target on every push to a branch",
		Long: `Deploy this bundle target on every push to a branch.

Creates the auto-deploy trigger for the target, or updates it if the target
already has one. A new trigger also deploys the branch's latest commit right
away.

The trigger is for the bundle's origin URL, its path in the repository and the
target. The Git provider is inferred for github.com and dev.azure.com. For any
other host, pass --git-provider.

Examples:
  databricks bundle auto-deploy create -t prod
  databricks bundle auto-deploy create -t prod --branch release --deploy-as 692bc6d0-ffa3-11ed-be56-0242ac120002`,
		Args: root.NoArgs,
	}

	var branch string
	var deployAs string
	var providerFlag gitwebhooks.GitProvider
	cmd.Flags().StringVar(&branch, "branch", "", "Branch whose pushes deploy the target. Defaults to bundle.git.branch.")
	cmd.Flags().StringVar(&deployAs, "deploy-as", "", "Application ID of the service principal to deploy as, or your own email. Defaults to you.")
	cmd.Flags().Var(&providerFlag, "git-provider", "Git provider of the repository, for hosts other than github.com and dev.azure.com.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Whether the configuration sets the branch, before Initialize fills in the current one.
		var branchConfigured bool
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			InitFunc: func(b *bundle.Bundle) {
				branchConfigured = b.Config.Bundle.Git.Branch != ""
			},
		})
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		if override := localVariableOverride(ctx, cmd, b); override != "" {
			return fmt.Errorf("auto-deploy deploys with the variable values in the configuration only, but %s overrides them here. Set the values as variable defaults or target overrides instead", override)
		}

		t, err := resolveTrigger(b)
		if err != nil {
			return err
		}
		provider, err := resolveProvider(t.repoURL, providerFlag)
		if err != nil {
			return err
		}

		configuredBranch := b.Config.Bundle.Git.Branch
		if branch == "" {
			branch = configuredBranch
		}
		if branch == "" {
			return errors.New("cannot determine the branch to deploy from. Pass it with --branch")
		}
		// A deploy fails when the pushed branch is not the configured one.
		if branchConfigured && branch != configuredBranch {
			return fmt.Errorf("git.branch is %s for target %s, so a deploy from %s would fail. Omit --branch, or change git.branch", configuredBranch, t.target, branch)
		}

		api, err := gitwebhooks.New(b.WorkspaceClient(ctx))
		if err != nil {
			return err
		}
		resp, err := api.UpsertWebhookTrigger(ctx, gitwebhooks.UpsertWebhookTrigger{
			RepoURL:     t.repoURL,
			GitProvider: provider,
			TriggerConfig: &gitwebhooks.DeploymentTriggerConfig{
				PushEvent: &gitwebhooks.PushEventConfig{Branch: branch},
				DabsDeploy: &gitwebhooks.DabsDeployTarget{
					BundleYmlRepoPath: t.bundlePath,
					DeploymentMode:    t.target,
					DeployAs:          deployAsIdentity(deployAs),
				},
			},
		})
		if err != nil {
			return err
		}

		if root.OutputType(cmd) == flags.OutputJSON {
			return cmdio.Render(ctx, resp)
		}
		identity := deployAs
		if identity == "" {
			identity = "you"
		}
		cmdio.LogString(ctx, fmt.Sprintf("Pushes to branch %s now deploy target %s as %s.\nTrigger ID: %s", branch, t.target, identity, resp.TriggerID))
		return nil
	}

	return cmd
}

// localVariableOverride names the --var flag, environment variable or overrides file that sets a
// variable here, or returns "". The runner deploys from a clone, which has none of them.
func localVariableOverride(ctx context.Context, cmd *cobra.Command, b *bundle.Bundle) string {
	if cmd.Flags().Changed("var") {
		return "--var"
	}
	for _, name := range slices.Sorted(maps.Keys(b.Config.Variables)) {
		if _, ok := env.Lookup(ctx, mutator.VariableEnvPrefix+name); ok {
			return mutator.VariableEnvPrefix + name
		}
	}
	overrides := mutator.VariableOverridesPath(b.Config.Bundle.Target)
	if _, err := os.Stat(filepath.Join(b.BundleRootPath, overrides)); err == nil {
		return overrides
	}
	return ""
}

// deployAsIdentity returns nil for an empty deployAs, so the service deploys as the caller.
func deployAsIdentity(deployAs string) *gitwebhooks.DeployAsIdentity {
	if deployAs == "" {
		return nil
	}
	// A service principal's application ID is a UUID.
	if iamutil.IsServicePrincipalName(deployAs) {
		return &gitwebhooks.DeployAsIdentity{ServicePrincipalClientID: deployAs}
	}
	return &gitwebhooks.DeployAsIdentity{UserEmail: deployAs}
}
