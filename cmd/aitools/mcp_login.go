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

// mcpLoginCommand is the command that signs a profile in to a service. The hints name it.
const (
	mcpLoginCommand  = "databricks aitools mcp login"
	mcpLogoutCommand = "databricks aitools mcp logout"
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

	// alreadySignedIn is set by login when the caller was signed in before it ran.
	alreadySignedIn bool
}

// mcpSignInAction runs login or logout against the profile's client and records the result in out. pinnedFile is the
// config file the server is pinned to, or "" (see mcpLoginHint).
type mcpSignInAction func(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, out *mcpSignInOutput, pinnedFile string) error

// NewMcpLoginCmd returns `aitools mcp login`.
func NewMcpLoginCmd() *cobra.Command {
	var auth *telemetryAuth
	cmd := &cobra.Command{
		Use:   "login <service>",
		Short: "Sign in to a Databricks managed MCP service with your account",
		Long: `Sign in to a Databricks managed MCP service that uses your account with an external provider, such
as GitHub. The sign-in page opens in the browser. The command waits for up to five minutes for the
sign-in to complete. Without a browser, open the printed URL yourself.

<service> is a full name such as system.ai.github. The profile is the one whose user signs in.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			auth.waitAtExit = true
			return runMcpSignInCommand(cmd, args[0], mcpLogin, protos.AitoolsMcpOperationSignIn)
		},
	}
	auth = withTelemetryAuth(cmd)
	return cmd
}

// NewMcpLogoutCmd returns `aitools mcp logout`.
func NewMcpLogoutCmd() *cobra.Command {
	var auth *telemetryAuth
	cmd := &cobra.Command{
		Use:   "logout <service>",
		Short: "Remove your sign-in to a Databricks managed MCP service",
		Long: `Remove your sign-in to a Databricks managed MCP service on a profile. The install of the server
stays in place. Running it when you are not signed in is not an error.

<service> is a full name such as system.ai.github.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			auth.waitAtExit = true
			return runMcpSignInCommand(cmd, args[0], mcpLogout, protos.AitoolsMcpOperationSignOut)
		},
	}
	auth = withTelemetryAuth(cmd)
	return cmd
}

// runMcpSignInCommand resolves the profile for service and runs action. In JSON mode the document is
// the only stdout output; progress goes to stderr.
func runMcpSignInCommand(cmd *cobra.Command, service string, action mcpSignInAction, operation protos.AitoolsMcpOperation) error {
	ctx := cmd.Context()
	jsonMode := installOutputIsJSON(cmd)

	out := &mcpSignInOutput{Service: service}
	runErr := runMcpSignIn(ctx, cmd.ErrOrStderr(), profileFlag(cmd), service, action, out)
	// The host is set once the profile's binding has passed and its client exists, so a run that reached no server logs nothing.
	if out.Host != "" {
		logMcpEvent(ctx, signInEvent(operation, out, runErr))
	}
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

// runMcpSignIn checks the service name, resolves the profile and checks its binding to the server, then runs action with
// the profile's client. out records the outcome: the profile is set once it is resolved, and the host once the client
// exists. Its errors are returned for the command to print.
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

// signInEvent is the SIGN_IN or SIGN_OUT event of one login or logout.
func signInEvent(operation protos.AitoolsMcpOperation, out *mcpSignInOutput, err error) protos.AitoolsMcpEvent {
	var r protos.AitoolsMcpServerResult
	if operation == protos.AitoolsMcpOperationSignIn {
		r = signInServerResult(out.Service, out.State, out.alreadySignedIn, err)
	} else {
		r = signOutServerResult(out.Service, out.State, err)
	}
	return protos.AitoolsMcpEvent{
		CallingAgent: callingAgentType(),
		Operation:    operation,
		Servers:      []protos.AitoolsMcpServerResult{r},
	}
}

// serviceFor returns the service with its sign-in requirement, as the profile sees it. A service the profile cannot see
// is an error, so that a misspelled name is reported rather than treated as a service with no sign-in.
func serviceFor(ctx context.Context, w *databricks.WorkspaceClient, profileName, pinnedFile, service string) (mcp.Service, error) {
	svc, err := mcp.GetService(ctx, w, service)
	if mcp.IsNotFound(err) {
		return mcp.Service{}, failure(protos.AitoolsErrorCategoryMCPServiceNotFound, err, "%s is not an MCP service visible to profile %s", service, profileName)
	}
	if err != nil {
		return mcp.Service{}, describeRequestError(profileName, pinnedFile, err)
	}
	return svc, nil
}

// mcpLogin signs the profile in to the service, when the service needs sign-in. A service that does not need it is
// NOT_REQUIRED and nothing is done. The sign-in URL is the one for the workspace the profile is in now.
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
	state, alreadySignedIn, urlShown, err := signIn(ctx, errOut, w, pinnedFile, url, svc, signInOptions{Profile: out.Profile, Open: browser.OpenOutputToStderr})
	out.State = state
	out.alreadySignedIn = alreadySignedIn
	// The URL is named when it was printed. A sign-in that failed before its page is printed names none.
	if urlShown {
		out.URL = url
	}
	return err
}

// mcpLogout removes the profile's sign-in to the service. The state is DELETED, or NOT_SIGNED_IN when there was no
// sign-in, which is not an error. A service the profile cannot see is an error, since a misspelled name looks the same.
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

// mcpLoginHint returns the command that signs profileName in to service. pinnedFile is the config file the server is
// pinned to, which the command sets, or "" for the file the shell reads.
func mcpLoginHint(service, profileName, pinnedFile string) string {
	return mcp.ConfigFileCommand(pinnedFile, mcpLoginCommand+" "+shellquote.BashArg(service)+" --profile "+shellquote.BashArg(profileName))
}

// mcpLogoutHint returns the command that removes profileName's sign-in to service. pinnedFile is as in mcpLoginHint.
func mcpLogoutHint(service, profileName, pinnedFile string) string {
	return mcp.ConfigFileCommand(pinnedFile, mcpLogoutCommand+" "+shellquote.BashArg(service)+" --profile "+shellquote.BashArg(profileName))
}

// lastCheckText names the failure of the last poll of a sign-in that timed out, for the timeout message. It is empty
// when the last poll read the credential.
func lastCheckText(timeout *mcp.SignInTimeoutError) string {
	if timeout.LastCheck == nil {
		return ""
	}
	return " (the last check failed: " + timeout.LastCheck.Error() + ")"
}

// signInPollInterval and signInWaitTimeout are the polling settings of a sign-in. Tests shorten them.
var (
	signInPollInterval = mcp.SignInPollInterval
	signInWaitTimeout  = mcp.SignInWaitTimeout
)

// signInOptions is what a caller of signIn sets: the profile named in the notice before the page, the page opener, and
// the poll callback. signIn adds the polling settings.
type signInOptions struct {
	Profile string
	Open    func(ctx context.Context, url string) error
	OnPoll  func()
}

// signIn runs the provider sign-in of svc for the profile named in opts and reports progress on errOut. It returns the
// sign-in state name, whether the caller was already signed in, whether url was printed, and the failure, which is
// categorized for JSON. `add --login` shares this.
func signIn(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, pinnedFile, url string, svc mcp.Service, opts signInOptions) (state string, alreadySignedIn, urlShown bool, err error) {
	profileName := opts.Profile
	alreadySignedIn, urlShown, err = mcp.SignIn(ctx, w, url, svc.FullName, errOut, mcp.SignInOptions{
		Profile: profileName,
		Poll:    signInPollInterval,
		Timeout: signInWaitTimeout,
		Open:    opts.Open,
		OnPoll:  opts.OnPoll,
	})
	timeout, timedOut := errors.AsType[*mcp.SignInTimeoutError](err)
	switch {
	case timedOut:
		return signInStateTimeout, false, urlShown, failure(protos.AitoolsErrorCategorySignInTimeout, err,
			"sign-in to %s did not complete in %g minutes%s. Finish at %s then retry: %s",
			svc.FullName, signInWaitTimeout.Minutes(), lastCheckText(timeout), url, mcpLoginHint(svc.FullName, profileName, pinnedFile))
	case errors.Is(err, mcp.ErrSignInFailed):
		return signInStateFailed, false, urlShown, failure(protos.AitoolsErrorCategoryUncategorized, err,
			"sign-in to %s failed on profile %s. Run: %s",
			svc.FullName, profileName, mcpLoginHint(svc.FullName, profileName, pinnedFile))
	case err != nil:
		return "", false, urlShown, describeRequestError(profileName, pinnedFile, err)
	}

	reportSignedIn(errOut, svc, profileName, alreadySignedIn)
	return signInStateActive, alreadySignedIn, urlShown, nil
}

// reportSignedIn prints that the profile is signed in to svc, and the note that the service has for its users.
func reportSignedIn(errOut io.Writer, svc mcp.Service, profileName string, alreadySignedIn bool) {
	if alreadySignedIn {
		fmt.Fprintf(errOut, "Already signed in to %s on profile %s.\n", svc.FullName, profileName)
	} else {
		fmt.Fprintf(errOut, "Signed in to %s on profile %s.\n", svc.FullName, profileName)
	}
	if svc.Known != nil && svc.Known.Note != "" {
		fmt.Fprintln(errOut, svc.Known.Note)
	}
}
