package apps

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// The apps create/update/deploy commands accept a git repository and a git
// deployment source, but the code generator emits these nested objects as
// `// TODO: complex arg` and only exposes them via --json. These overrides add
// ergonomic top-level flags for the git fields so users can point an app at a
// repo and deploy a specific ref without hand-writing JSON.
//
// The SDK models GitRepository and GitSource as optional pointers on the
// request. We leave them nil unless the user sets a git flag; allocating them
// unconditionally would send an empty object on every non-git create/deploy and
// change the request the server sees.

// gitRepositoryFlags binds the git repository flags onto an *apps.GitRepository
// pointer field (App.GitRepository), used by both create and update. url and
// provider identify the repository; auto-deploy is a modifier on it
// (push-to-deploy). withCallerCredential adds --caller-git-credential-id (the
// credential granting the app's service principal repo access); the server
// rejects it on update, so only create passes true. It returns a PreRunE that
// allocates the struct only when a flag was set.
func gitRepositoryFlags(cmd *cobra.Command, target **apps.GitRepository, withCallerCredential bool) func(*cobra.Command, []string) error {
	var url, provider string
	var autoDeploy bool
	var callerCredentialID int64
	cmd.Flags().StringVar(&url, "git-url", "", "URL of the Git repository the app deploys from.")
	cmd.Flags().StringVar(&provider, "git-provider", "", "Git provider. Case insensitive. Supported values: gitHub, gitHubEnterprise, bitbucketCloud, bitbucketServer, azureDevOpsServices, gitLab, gitLabEnterpriseEdition, awsCodeCommit.")
	cmd.Flags().BoolVar(&autoDeploy, "git-auto-deploy", false, "Automatically redeploy the app on pushes to the configured Git branch. Requires --git-url and --git-provider.")

	// caller_credential_id can only be set on create; the server rejects it on
	// update. The modifier list in the "requires repo" error tracks this so the
	// message never names a flag the command doesn't have.
	modifiers := "--git-auto-deploy"
	if withCallerCredential {
		cmd.Flags().Int64Var(&callerCredentialID, "caller-git-credential-id", 0, "ID of the caller's Git credential used to grant the app's service principal access to the repository. Create only. Requires --git-url and --git-provider.")
		modifiers = "--git-auto-deploy and --caller-git-credential-id"
	}

	return func(cmd *cobra.Command, args []string) error {
		urlSet := cmd.Flags().Changed("git-url")
		providerSet := cmd.Flags().Changed("git-provider")
		autoDeploySet := cmd.Flags().Changed("git-auto-deploy")
		credSet := withCallerCredential && cmd.Flags().Changed("caller-git-credential-id")
		if !urlSet && !providerSet && !autoDeploySet && !credSet {
			return nil
		}
		// The server requires both url and provider together, so fail early
		// rather than shipping a half-populated repository it will reject.
		if urlSet != providerSet {
			return errors.New("--git-url and --git-provider must be set together")
		}
		// auto-deploy and the caller credential are modifiers on the repository;
		// without url+provider there is no repository to attach them to.
		if (autoDeploySet || credSet) && !urlSet {
			return errors.New(modifiers + " require --git-url and --git-provider")
		}
		repo := &apps.GitRepository{Url: url, Provider: provider}
		if autoDeploySet {
			repo.AutoDeploy = autoDeploy
			// auto_deploy is omitempty; force-send so --git-auto-deploy=false is
			// transmitted (e.g. to turn auto-deploy off on update).
			repo.ForceSendFields = append(repo.ForceSendFields, "AutoDeploy")
		}
		if credSet {
			repo.CallerCredentialId = callerCredentialID
		}
		*target = repo
		return nil
	}
}

// gitSourceFlags binds the git source flags onto an *apps.GitSource pointer
// field: AppDeployment.GitSource on deploy (the ref for one deployment), and
// App.GitSource on create/update (the app's default deployment source, echoed
// back as default_git_source). It returns a PreRunE that allocates the struct
// only when a flag was set.
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

// requireBranchForAutoDeploy enforces that enabling auto-deploy is paired with a
// branch: auto-deploy redeploys on pushes to a branch, so a tag, a commit, or no
// ref at all has nothing to watch. Turning auto-deploy off needs no branch. Both
// create and update set the branch explicitly here rather than letting the
// server fall back to the app's existing default_git_source. The check is a
// no-op on any command without a --git-branch flag to satisfy it.
func requireBranchForAutoDeploy(cmd *cobra.Command, args []string) error {
	if !cmd.Flags().Changed("git-auto-deploy") || cmd.Flags().Lookup("git-branch") == nil {
		return nil
	}
	on, err := cmd.Flags().GetBool("git-auto-deploy")
	if err != nil {
		return err
	}
	if on && !cmd.Flags().Changed("git-branch") {
		return errors.New("--git-auto-deploy requires --git-branch: auto-deploy redeploys on pushes to a branch")
	}
	return nil
}

// requireRepoForGitSourceOnCreate enforces that a create-time git source is
// paired with a repository: App.GitSource resolves its ref against
// App.GitRepository, and on create there is no prior repository to fall back on
// (unlike update/deploy, which reference the app's already-configured repo).
// gitRepositoryFlags runs first and rejects url without provider, so checking
// --git-url alone implies both are set.
func requireRepoForGitSourceOnCreate(cmd *cobra.Command, args []string) error {
	srcSet := cmd.Flags().Changed("git-branch") ||
		cmd.Flags().Changed("git-tag") ||
		cmd.Flags().Changed("git-commit") ||
		cmd.Flags().Changed("git-source-code-path")
	if srcSet && !cmd.Flags().Changed("git-url") {
		return errors.New("--git-branch, --git-tag, --git-commit, and --git-source-code-path require --git-url and --git-provider on create: the git source resolves against the app's repository")
	}
	return nil
}

func gitCreateOverride(createCmd *cobra.Command, createReq *apps.CreateAppRequest) {
	chainPreRunE(createCmd, gitRepositoryFlags(createCmd, &createReq.App.GitRepository, true))
	// App.GitSource is the create-time deployment source (the default ref);
	// the server persists it and reports it back as default_git_source.
	chainPreRunE(createCmd, gitSourceFlags(createCmd, &createReq.App.GitSource))
	chainPreRunE(createCmd, requireBranchForAutoDeploy)
	chainPreRunE(createCmd, requireRepoForGitSourceOnCreate)
}

// applyPreservedAutoDeploy carries a currently-enabled auto_deploy forward onto
// the update request's git_repository. auto_deploy=true survives the omitempty
// tag, so no ForceSendFields entry is needed; a currently-disabled auto_deploy
// stays omitted (the server keeps it off).
func applyPreservedAutoDeploy(repo *apps.GitRepository, currentAutoDeploy bool) {
	if repo == nil || !currentAutoDeploy {
		return
	}
	repo.AutoDeploy = true
}

// preserveAutoDeployOnUpdate keeps auto-deploy from being silently torn down
// when a flag-based update changes the repository without re-asserting it. It
// runs only on the flag path (not --json, which owns the whole body), only when
// a repository change is being sent, and only when the user did not pass
// --git-auto-deploy themselves; in that case it reads the app's current
// auto_deploy from GetApp and carries it forward.
func preserveAutoDeployOnUpdate(req *apps.UpdateAppRequest) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") || cmd.Flags().Changed("git-auto-deploy") {
			return nil
		}
		if req.App.GitRepository == nil || len(args) == 0 {
			return nil
		}
		ctx := cmd.Context()
		current, err := cmdctx.WorkspaceClient(ctx).Apps.GetByName(ctx, args[0])
		if err != nil {
			return fmt.Errorf("read current app to preserve auto-deploy: %w", err)
		}
		if current.GitRepository != nil {
			applyPreservedAutoDeploy(req.App.GitRepository, current.GitRepository.AutoDeploy)
		}
		return nil
	}
}

func gitUpdateOverride(updateCmd *cobra.Command, updateReq *apps.UpdateAppRequest) {
	chainPreRunE(updateCmd, gitRepositoryFlags(updateCmd, &updateReq.App.GitRepository, false))
	// App.GitSource lets update set the deployment source (e.g. the branch
	// auto-deploy watches) explicitly instead of leaning on the app's existing
	// default_git_source.
	chainPreRunE(updateCmd, gitSourceFlags(updateCmd, &updateReq.App.GitSource))
	chainPreRunE(updateCmd, requireBranchForAutoDeploy)
	// Runs last: preserve auto-deploy across a repository change the user made
	// without re-asserting --git-auto-deploy.
	chainPreRunE(updateCmd, preserveAutoDeployOnUpdate(updateReq))
}

func gitDeployOverride(deployCmd *cobra.Command, deployReq *apps.CreateAppDeploymentRequest) {
	chainPreRunE(deployCmd, gitSourceFlags(deployCmd, &deployReq.AppDeployment.GitSource))
}

func init() {
	createOverrides = append(createOverrides, gitCreateOverride)
	updateOverrides = append(updateOverrides, gitUpdateOverride)
	deployOverrides = append(deployOverrides, gitDeployOverride)
}
