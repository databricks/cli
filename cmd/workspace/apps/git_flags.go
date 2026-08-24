package apps

import (
	"errors"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// The apps create/update/deploy commands accept a git repository and a git
// deployment source, but the code generator emits these nested objects as
// `// TODO: complex arg` and only exposes them via --json. These overrides add
// ergonomic top-level flags for the GA git fields so users can point an app at
// a repo and deploy a specific ref without hand-writing JSON.
//
// The SDK models GitRepository and GitSource as optional pointers on the
// request. We leave them nil unless the user sets a git flag; allocating them
// unconditionally would send an empty object on every non-git create/deploy and
// change the request the server sees.

// gitRepositoryFlags binds --git-url/--git-provider onto an *apps.GitRepository
// pointer field (App.GitRepository), used by both create and update. It returns
// a PreRunE that allocates the struct only when a flag was set.
func gitRepositoryFlags(cmd *cobra.Command, target **apps.GitRepository) func(*cobra.Command, []string) error {
	var url, provider string
	cmd.Flags().StringVar(&url, "git-url", "", "URL of the Git repository the app deploys from.")
	cmd.Flags().StringVar(&provider, "git-provider", "", "Git provider. Case insensitive. Supported values: gitHub, gitHubEnterprise, bitbucketCloud, bitbucketServer, azureDevOpsServices, gitLab, gitLabEnterpriseEdition, awsCodeCommit.")

	return func(cmd *cobra.Command, args []string) error {
		urlSet := cmd.Flags().Changed("git-url")
		providerSet := cmd.Flags().Changed("git-provider")
		if !urlSet && !providerSet {
			return nil
		}
		// The server requires both url and provider together, so fail early
		// rather than shipping a half-populated repository it will reject.
		if urlSet != providerSet {
			return errors.New("--git-url and --git-provider must be set together")
		}
		*target = &apps.GitRepository{Url: url, Provider: provider}
		return nil
	}
}

// gitSourceFlags binds the deploy-time git source flags onto an *apps.GitSource
// pointer field (AppDeployment.GitSource). It returns a PreRunE that allocates
// the struct only when a flag was set.
func gitSourceFlags(cmd *cobra.Command, target **apps.GitSource) func(*cobra.Command, []string) error {
	var branch, tag, commit, sourceCodePath string
	cmd.Flags().StringVar(&branch, "git-branch", "", "Git branch to deploy from.")
	cmd.Flags().StringVar(&tag, "git-tag", "", "Git tag to deploy from.")
	cmd.Flags().StringVar(&commit, "git-commit", "", "Git commit SHA to deploy from.")
	cmd.Flags().StringVar(&sourceCodePath, "git-source-code-path", "", "Relative path to the app source code within the Git repository. Defaults to the repository root.")

	// branch, tag, and commit are a proto oneof (a single git reference) — the
	// server accepts at most one.
	cmd.MarkFlagsMutuallyExclusive("git-branch", "git-tag", "git-commit")

	return func(cmd *cobra.Command, args []string) error {
		refSet := cmd.Flags().Changed("git-branch") ||
			cmd.Flags().Changed("git-tag") ||
			cmd.Flags().Changed("git-commit")
		pathSet := cmd.Flags().Changed("git-source-code-path")
		if !refSet && !pathSet {
			return nil
		}
		// deployment_source is a proto oneof: a deployment draws its code from
		// either a workspace path (--source-code-path) or a Git source, never
		// both. Reject the combination rather than shipping a request the server
		// and bundle validation treat as invalid.
		if cmd.Flags().Changed("source-code-path") {
			return errors.New("--source-code-path (workspace source) cannot be combined with the --git-* flags; a deployment uses either a workspace path or a Git source")
		}
		// A source-code path without a reference has no repository to resolve
		// against — the reference is what selects the code to deploy.
		if pathSet && !refSet {
			return errors.New("--git-source-code-path requires one of --git-branch, --git-tag, or --git-commit")
		}
		*target = &apps.GitSource{
			Branch:         branch,
			Tag:            tag,
			Commit:         commit,
			SourceCodePath: sourceCodePath,
		}
		return nil
	}
}

// chainPreRunE runs fn after any PreRunE already set on the command, preserving
// the generated cmd.PreRunE (e.g. root.MustWorkspaceClient) rather than
// replacing it.
func chainPreRunE(cmd *cobra.Command, fn func(*cobra.Command, []string) error) {
	prev := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if prev != nil {
			if err := prev(cmd, args); err != nil {
				return err
			}
		}
		return fn(cmd, args)
	}
}

func gitCreateOverride(createCmd *cobra.Command, createReq *apps.CreateAppRequest) {
	chainPreRunE(createCmd, gitRepositoryFlags(createCmd, &createReq.App.GitRepository))
}

func gitUpdateOverride(updateCmd *cobra.Command, updateReq *apps.UpdateAppRequest) {
	chainPreRunE(updateCmd, gitRepositoryFlags(updateCmd, &updateReq.App.GitRepository))
}

func gitDeployOverride(deployCmd *cobra.Command, deployReq *apps.CreateAppDeploymentRequest) {
	chainPreRunE(deployCmd, gitSourceFlags(deployCmd, &deployReq.AppDeployment.GitSource))
}

func init() {
	createOverrides = append(createOverrides, gitCreateOverride)
	updateOverrides = append(updateOverrides, gitUpdateOverride)
	deployOverrides = append(deployOverrides, gitDeployOverride)
}
