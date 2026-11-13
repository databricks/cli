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
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/spf13/cobra"
)

// categorySignInTimeout is the error category of a sign-in that did not finish in time. The
// telemetry enum does not define it yet.
const categorySignInTimeout protos.AitoolsErrorCategory = "SIGN_IN_TIMEOUT"

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
	Service       string                      `json:"service"`
	Profile       string                      `json:"profile"`
	Host          string                      `json:"host"`
	State         string                      `json:"state"`
	URL           string                      `json:"url,omitempty"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`
	Error         string                      `json:"error,omitempty"`

	// configFile is the resolved config file that the profile is read from. It is not in the JSON.
	configFile string
}

// mcpSignInAction runs login or logout against the profile's client and records the result in out.
type mcpSignInAction func(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput) error

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
		return err
	}
	if runErr != nil {
		return root.ErrAlreadyPrinted
	}
	return nil
}

func runMcpSignIn(ctx context.Context, errOut io.Writer, flagValue, service string, action mcpSignInAction, out *mcpSignInOutput) error {
	if !isFullName(strings.Split(service, ".")) {
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
	// The binding is checked before the profile's host is contacted, as add and serve check it.
	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	if _, err := mcp.CheckBinding(ctx, state, service, profileName, configFile); err != nil {
		return err
	}
	w, err := profileClient(ctx, profileName, configFile)
	if err != nil {
		return err
	}
	out.Profile = profileName
	out.Host = w.Config.Host
	out.configFile = configFile
	return action(ctx, errOut, w, out)
}

// signInService returns the service of a sign-in command. A service the profile cannot see is an error, so that a
// misspelled name is reported rather than treated as a service with no sign-in.
func signInService(ctx context.Context, w *databricks.WorkspaceClient, out *mcpSignInOutput) (mcp.Service, error) {
	svc, err := mcp.GetService(ctx, w, out.Service)
	if errors.Is(err, apierr.ErrNotFound) {
		return mcp.Service{}, failure(categoryServiceNotFound, err, "%s is not an MCP service visible to profile %s", out.Service, out.Profile)
	}
	if err != nil {
		return mcp.Service{}, describeRequestError(out.Profile, err)
	}
	return svc, nil
}

func mcpLogin(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput) error {
	svc, err := signInService(ctx, w, out)
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
		return describeRequestError(out.Profile, fmt.Errorf("cannot read the workspace ID of %s: %w", w.Config.Host, err))
	}
	out.URL = mcp.SignInURL(out.Host, svc.FullName, workspaceID)
	state, err := signIn(ctx, errOut, w, out.Profile, out.URL, svc)
	out.State = state
	return err
}

func mcpLogout(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput) error {
	if _, err := signInService(ctx, w, out); err != nil {
		return err
	}
	deleted, err := mcp.SignOut(ctx, w, out.Service)
	if err != nil {
		return describeRequestError(out.Profile, err)
	}
	if !deleted {
		out.State = signOutStateNotSignedIn
		fmt.Fprintf(errOut, "Not signed in to %s on profile %s.\n", out.Service, out.Profile)
		return nil
	}
	out.State = signOutStateDeleted
	fmt.Fprintf(errOut, "Signed out of %s on profile %s.\n", out.Service, out.Profile)
	return nil
}

// signIn runs the provider sign-in of svc for the profile and reports progress on errOut. It returns
// the sign-in state name and the failure, which is categorized for JSON. `add --login` shares it.
func signIn(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, profileName, url string, svc mcp.Service) (string, error) {
	fmt.Fprintf(errOut, "Sign in with the same account you use for Databricks profile %s.\n", profileName)
	err := mcp.SignIn(ctx, w, url, svc.FullName, errOut, mcp.SignInOptions{
		Poll:    mcp.SignInPollInterval,
		Timeout: mcp.SignInWaitTimeout,
		Open:    browser.Open,
	})
	switch {
	case errors.Is(err, mcp.ErrSignInTimeout):
		return signInStateTimeout, failure(categorySignInTimeout, err,
			"Sign-in to %s did not complete in %g minutes. Finish at %s then retry: databricks aitools mcp login %s --profile %s",
			svc.FullName, mcp.SignInWaitTimeout.Minutes(), url, svc.FullName, profileName)
	case errors.Is(err, mcp.ErrSignInFailed):
		return signInStateFailed, failure(protos.AitoolsErrorCategoryUncategorized, err,
			"Sign-in to %s failed on profile %s. Run: databricks aitools mcp login %s --profile %s",
			svc.FullName, profileName, svc.FullName, profileName)
	case err != nil:
		return "", describeRequestError(profileName, err)
	}

	fmt.Fprintf(errOut, "Signed in to %s on profile %s.\n", svc.FullName, profileName)
	if svc.Known != nil && svc.Known.Note != "" {
		fmt.Fprintln(errOut, svc.Known.Note)
	}
	return signInStateActive, nil
}
