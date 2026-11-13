package aitools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

// Sign-in states of `mcp login` and `mcp logout`, reported in JSON.
const (
	signInStateActive       = "ACTIVE"
	signInStateTimeout      = "TIMEOUT"
	signInStateFailed       = "FAILED"
	signInStateNotRequired  = "NOT_REQUIRED"
	signOutStateDeleted     = "DELETED"
	signOutStateNotSignedIn = "NOT_SIGNED_IN"
)

// mcpSignInOutput is the `mcp login` and `mcp logout` document.
type mcpSignInOutput struct {
	Service string `json:"service"`
	// Profile and Host are set once they are known, so a failure before that leaves them out.
	Profile       string                      `json:"profile,omitempty"`
	Host          string                      `json:"host,omitempty"`
	State         string                      `json:"state,omitempty"`
	URL           string                      `json:"url,omitempty"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`
	Error         string                      `json:"error,omitempty"`
}

// mcpSignInAction runs login or logout against the profile's client and records the result in out. pinnedFile is the
// config file the server is pinned to, or "" (see mcpLoginHint).
type mcpSignInAction func(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput, pinnedFile string) error

// NewMcpLoginCmd returns `aitools mcp login`.
func NewMcpLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <service>",
		Short: "Sign in to a Databricks managed MCP service with your account",
		Long: `Sign in to a Databricks managed MCP service that uses your account with an external provider, such
as GitHub. The sign-in page opens in the browser. The command waits for up to five minutes for the
sign-in to complete. Without a browser, open the printed URL yourself.

<service> is a full name such as system.ai.github. The profile is the one whose user signs in.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMcpSignInCommand(cmd, args[0], mcpLogin)
		},
	}
}

// NewMcpLogoutCmd returns `aitools mcp logout`.
func NewMcpLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout <service>",
		Short: "Remove your sign-in to a Databricks managed MCP service",
		Long: `Remove your sign-in to a Databricks managed MCP service on a profile. The install of the server
stays in place. Running it when you are not signed in is not an error.

<service> is a full name such as system.ai.github.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMcpSignInCommand(cmd, args[0], mcpLogout)
		},
	}
}

// runMcpSignInCommand resolves the profile for service and runs action. In JSON mode the document is
// the only stdout output; progress goes to stderr.
func runMcpSignInCommand(cmd *cobra.Command, service string, action mcpSignInAction) error {
	ctx := cmd.Context()
	jsonMode := installOutputIsJSON(cmd)

	out := &mcpSignInOutput{Service: service}
	runErr := runMcpSignIn(ctx, cmd.ErrOrStderr(), profileFlag(cmd), service, action, out)
	if !jsonMode {
		return runErr
	}
	if runErr != nil {
		out.Error = runErr.Error()
		out.ErrorCategory = errorCategory(runErr)
	}
	if err := renderJSON(cmd.OutOrStdout(), out); err != nil {
		return errors.Join(runErr, err)
	}
	if runErr != nil {
		return root.ErrAlreadyPrinted
	}
	return nil
}

// isServiceName reports whether service is a full name that can be sent in a request path as it is.
func isServiceName(service string) bool {
	return isFullName(strings.Split(service, ".")) && mcp.PathSafeName(service)
}

func runMcpSignIn(ctx context.Context, errOut io.Writer, flagValue, service string, action mcpSignInAction, out *mcpSignInOutput) error {
	if !isServiceName(service) {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "service must be a full name such as system.ai.github, got %q", service)
	}
	// The config file is resolved once, as add resolves it, so that the profile lookup and the client read the same file.
	configFile, _, err := addConfigFile(ctx)
	if err != nil {
		return err
	}
	ctx = env.Set(ctx, mcpconfig.ConfigFileEnv, configFile)
	profileName, err := resolveProfile(ctx, flagValue)
	if err != nil {
		return err
	}
	out.Profile = profileName
	// The binding is checked before the profile's host is contacted, as add and serve check it.
	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	entry, err := mcp.CheckBinding(ctx, state, service, profileName, configFile)
	if err != nil {
		return err
	}
	// A server pinned to a config file names that file in the hints, as add does.
	pinnedFile := mcp.PinnedFile(entry, configFile)
	w, err := profileClient(ctx, profileName, configFile, pinnedFile)
	if err != nil {
		return err
	}
	out.Host = w.Config.Host
	return action(ctx, errOut, w, out, pinnedFile)
}

// serviceFor returns the service with its sign-in requirement, as the profile sees it. A service the profile cannot see
// is an error, so that a misspelled name is reported rather than treated as a service with no sign-in.
func serviceFor(ctx context.Context, w *databricks.WorkspaceClient, profileName, pinnedFile, service string) (mcp.Service, error) {
	svc, err := mcp.GetService(ctx, w, service)
	if mcp.IsNotFound(err) {
		return mcp.Service{}, failure(categoryServiceNotFound, err, "%s is not an MCP service visible to profile %s", service, profileName)
	}
	if err != nil {
		return mcp.Service{}, describeRequestError(profileName, pinnedFile, err)
	}
	return svc, nil
}

func mcpLogin(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput, pinnedFile string) error {
	svc, err := serviceFor(ctx, w, out.Profile, pinnedFile, out.Service)
	if err != nil {
		return err
	}
	if !svc.SignInRequired {
		out.State = signInStateNotRequired
		fmt.Fprintf(errOut, "%s does not need sign-in.\n", svc.FullName)
		return nil
	}

	// The sign-in URL names the workspace the profile is in now, as add does. The binding check above has already
	// refused a profile whose workspace changed since the server was installed.
	workspaceID, err := auth.ResolveWorkspaceID(ctx, w)
	if err != nil {
		return describeRequestError(out.Profile, pinnedFile, fmt.Errorf("cannot read the workspace ID of %s: %w", w.Config.Host, err))
	}
	url := mcp.SignInURL(out.Host, svc.FullName, workspaceID)
	state, alreadySignedIn, err := signIn(ctx, errOut, w, out.Profile, pinnedFile, url, svc)
	out.State = state
	// The URL is named only when a page was opened for the sign-in, or when the sign-in failed and the URL is the way to
	// finish it.
	if !alreadySignedIn {
		out.URL = url
	}
	return err
}

func mcpLogout(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput, pinnedFile string) error {
	// The credential is deleted first, so that the credential of a service the profile can no longer see is removed too.
	// The service is looked up only when there is no credential, to tell a name that is not a service from a caller who
	// is not signed in.
	deleted, err := mcp.SignOut(ctx, w, out.Service)
	if err != nil {
		return describeRequestError(out.Profile, pinnedFile, err)
	}
	if deleted {
		out.State = signOutStateDeleted
		fmt.Fprintf(errOut, "Signed out of %s on profile %s.\n", out.Service, out.Profile)
		return nil
	}
	if _, err := serviceFor(ctx, w, out.Profile, pinnedFile, out.Service); err != nil {
		return err
	}
	out.State = signOutStateNotSignedIn
	fmt.Fprintf(errOut, "Not signed in to %s on profile %s.\n", out.Service, out.Profile)
	return nil
}

// mcpLoginCommand is the command that signs a profile in to a service. The hints name it.
const mcpLoginCommand = "databricks aitools mcp login"

// mcpLoginHint returns the command that signs profileName in to service. pinnedFile is the config file the server is
// pinned to, which the command sets, or "" for the file the shell reads.
func mcpLoginHint(service, profileName, pinnedFile string) string {
	return mcp.ConfigFileCommand(pinnedFile, mcpLoginCommand+" "+shellquote.BashArg(service)+" --profile "+shellquote.BashArg(profileName))
}

// signIn runs the provider sign-in of svc for the profile and reports progress on errOut. It returns the sign-in state
// name, whether the caller was already signed in, and the failure, which is categorized for JSON. `add --login` shares it.
func signIn(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, profileName, pinnedFile, url string, svc mcp.Service) (state string, alreadySignedIn bool, err error) {
	alreadySignedIn, err = mcp.SignIn(ctx, w, url, svc.FullName, errOut, mcp.SignInOptions{
		Profile: profileName,
		Poll:    mcp.SignInPollInterval,
		Timeout: mcp.SignInWaitTimeout,
		Open:    browser.OpenOutputToStderr,
	})
	switch {
	case errors.Is(err, mcp.ErrSignInTimeout):
		return signInStateTimeout, false, failure(categorySignInTimeout, err,
			"sign-in to %s did not complete in %g minutes. Finish at %s then retry: %s",
			svc.FullName, mcp.SignInWaitTimeout.Minutes(), url, mcpLoginHint(svc.FullName, profileName, pinnedFile))
	case errors.Is(err, mcp.ErrSignInFailed):
		return signInStateFailed, false, failure(protos.AitoolsErrorCategoryUncategorized, err,
			"sign-in to %s failed on profile %s. Run: %s",
			svc.FullName, profileName, mcpLoginHint(svc.FullName, profileName, pinnedFile))
	case err != nil:
		return "", false, describeRequestError(profileName, pinnedFile, err)
	}

	if alreadySignedIn {
		fmt.Fprintf(errOut, "Already signed in to %s on profile %s.\n", svc.FullName, profileName)
	} else {
		fmt.Fprintf(errOut, "Signed in to %s on profile %s.\n", svc.FullName, profileName)
	}
	if svc.Known != nil && svc.Known.Note != "" {
		fmt.Fprintln(errOut, svc.Known.Note)
	}
	return signInStateActive, alreadySignedIn, nil
}
