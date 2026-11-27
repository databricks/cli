package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// maxConfirmationAttempts is how many confirmation codes the user can enter
// for a single device code before the login fails.
const maxConfirmationAttempts = 3

// pendingLoginKeySuffix marks the token store entry that holds a
// non-interactive device code login until it completes. The entry sits next to
// the login's token, so that a token that still works is kept until then.
const pendingLoginKeySuffix = "#device-code"

// startNewLoginHint ends the errors of a non-interactive device code login
// that cannot be completed.
const startNewLoginHint = "Run the same command without --user-code and --confirmation-code to start a new login"

// deviceCodeAgentNote starts the note that tells an AI agent to pass the login
// on to the user.
const deviceCodeAgentNote = "give the URL and the code above to the user.\n" +
	"Do not open the URL or approve the login yourself: the user must approve it.\n"

// deviceCodeIncompatibleFlags lists flags that device code login cannot honor.
var deviceCodeIncompatibleFlags = []string{
	// Resource indicators are sent on the browser authorization request,
	// which the device code flow does not make.
	"resource",
}

var (
	errDeviceCodeNeedsHost = errors.New("--use-device-code requires a host. Pass it as an argument or with --host")
	// The cluster picker is interactive.
	errDeviceCodeNonInteractiveCluster = errors.New("--configure-cluster is not supported by the non-interactive device code login. Set cluster_id on the profile after the login")
)

// matchDeviceCodeProfiles matches the profiles that device code login can
// use: those whose login is not scoped to resources.
func matchDeviceCodeProfiles(p profile.Profile) bool {
	return len(u2mResourcesFromProfile(&p)) == 0
}

// validateDeviceCodeLogin returns an error if device code login is combined
// with flags it cannot honor.
func validateDeviceCodeLogin(cmd *cobra.Command) error {
	for _, name := range deviceCodeIncompatibleFlags {
		if cmd.Flag(name).Changed {
			return fmt.Errorf("--use-device-code cannot be combined with --%s", name)
		}
	}
	return nil
}

// deviceCodeLogin logs in using the Databricks variant of the RFC 8628
// device authorization grant. It shows the user where to approve the login
// and exchanges the device code together with the confirmation code that the
// user enters for a token. A rejected confirmation code leaves the device
// code valid, so the user can re-enter it up to maxConfirmationAttempts
// times.
func deviceCodeLogin(ctx context.Context, persistentAuth *u2m.PersistentAuth) (*oauth2.Token, error) {
	login, err := persistentAuth.NewDeviceCodeLogin(ctx)
	if err != nil {
		return nil, err
	}
	da, err := login.RequestDeviceAuthorization(ctx)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := range maxConfirmationAttempts {
		code, err := promptForConfirmationCode(ctx, *da, lastErr, maxConfirmationAttempts-attempt)
		if err != nil {
			return nil, err
		}

		token, err := login.RequestDeviceAccessToken(ctx, da.DeviceCode, code)
		if err == nil {
			return token, nil
		}
		lastErr = err
		if !errors.Is(err, u2m.ErrInvalidConfirmationCode) {
			return nil, fmt.Errorf("token exchange: %w", err)
		}
	}
	return nil, fmt.Errorf("%w. Start login from scratch", lastErr)
}

// validateDeviceCodeCompletionFlags returns an error if the flags that complete
// a non-interactive device code login are not set together, or are set to an
// empty code.
func validateDeviceCodeCompletionFlags(cmd *cobra.Command) error {
	useDeviceCode, err := cmd.Flags().GetBool("use-device-code")
	if err != nil {
		return err
	}
	hasUserCode := cmd.Flags().Changed("user-code")
	hasConfirmationCode := cmd.Flags().Changed("confirmation-code")
	switch {
	case hasConfirmationCode && !hasUserCode:
		return errors.New("--confirmation-code requires --user-code")
	case hasUserCode && !hasConfirmationCode:
		return errors.New("--user-code requires --confirmation-code")
	case hasConfirmationCode && !useDeviceCode:
		return errors.New("--confirmation-code requires --use-device-code")
	}
	for _, name := range []string{"user-code", "confirmation-code"} {
		if f := cmd.Flag(name); f.Changed && strings.TrimSpace(f.Value.String()) == "" {
			return fmt.Errorf("--%s cannot be empty", name)
		}
	}
	return nil
}

// nonInteractiveDeviceCodeLogin runs the two steps of a device code login
// that cannot read the confirmation code from stdin. The first step stores the
// device code as a pending login under key, and the second step exchanges it,
// so that the device code never leaves the machine. The host is canonical, like
// the host in the cache key, and the scopes are sorted, so that the same host
// or scopes passed in another form complete the login.
type nonInteractiveDeviceCodeLogin struct {
	persistentAuth *u2m.PersistentAuth
	tokenStore     storage.Store
	key            string
	profile        string
	host           string
	accountID      string
	clientID       string
	scopes         []string
}

// oauthAccountID returns the ID of the account whose OIDC endpoints arg logs
// in with, or "" for a workspace host.
func oauthAccountID(arg u2m.OAuthArgument) string {
	switch a := arg.(type) {
	case u2m.AccountOAuthArgument:
		return a.GetAccountId()
	case u2m.UnifiedOAuthArgument:
		return a.GetAccountId()
	}
	return ""
}

// pendingLoginKey returns the token store key of the pending login for the
// login with cacheKey.
func pendingLoginKey(cacheKey string) string {
	return cacheKey + pendingLoginKeySuffix
}

// validateLoginProfileName returns an error if the token of profile name would
// share its token store key with the pending login of another profile.
// It also rejects a name that would share a key with a part of another
// profile's access token.
func validateLoginProfileName(name string) error {
	if strings.HasSuffix(name, pendingLoginKeySuffix) {
		return fmt.Errorf("profile name %q must not end with %s, which marks pending device code logins", name, pendingLoginKeySuffix)
	}
	// Windows Credential Manager ignores case in credential names, so a
	// profile named "dev#Access-Token-0" would share the item of a part of dev.
	// https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw
	if strings.Contains(strings.ToLower(name), storage.AccessTokenPartMarker) {
		return fmt.Errorf("profile name %q must not contain %s, which marks the parts of large access tokens", name, storage.AccessTokenPartMarker)
	}
	return nil
}

// target describes the login in errors.
func (l nonInteractiveDeviceCodeLogin) target() string {
	if l.profile != "" {
		return "profile " + l.profile
	}
	return "host " + l.host
}

// start requests a device authorization, stores it as the pending login, and
// prints how to complete the login. A pending login under the same key is
// replaced.
func (l nonInteractiveDeviceCodeLogin) start(ctx context.Context) error {
	login, err := l.persistentAuth.NewDeviceCodeLogin(ctx)
	if err != nil {
		return err
	}
	da, err := login.RequestDeviceAuthorization(ctx)
	if err != nil {
		return err
	}
	err = l.tokenStore.Put(l.key, storage.Entry{PendingLogin: &storage.PendingLogin{
		DeviceCode: da.DeviceCode,
		UserCode:   da.UserCode,
		Host:       l.host,
		AccountID:  l.accountID,
		ClientID:   l.clientID,
		Scopes:     l.scopes,
	}})
	if err != nil {
		return fmt.Errorf("store pending login: %w", err)
	}
	agentNote := deviceCodeAgentNote + "Ask the user for the confirmation code their browser shows."
	cmdio.LogString(ctx, strings.Join(append(approvalInstructions(ctx, *da, agentNote),
		"Once finished, complete the login by running the same command with",
		"--user-code "+formatCode(da.UserCode)+" and --confirmation-code set to the confirmation code",
		"provided in your browser.",
	), "\n"))
	return nil
}

// complete exchanges the device code of the pending login started with
// userCode together with confirmationCode for a token.
func (l nonInteractiveDeviceCodeLogin) complete(ctx context.Context, userCode, confirmationCode string) (*oauth2.Token, error) {
	e, err := l.tokenStore.Lookup(l.key)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("read pending login: %w", err)
	}
	// The key can also hold the token of a profile that happens to be named
	// like a pending login key.
	pending := e.PendingLogin
	if pending == nil {
		return nil, fmt.Errorf("no pending device code login for %s. %s", l.target(), startNewLoginHint)
	}
	// The user code can be mistyped, or belong to a login that a newer one
	// replaced, because each profile or host has one pending login. Checking
	// the user code here avoids sending the confirmation code with the wrong
	// device code, which the server would reject as a wrong confirmation
	// code.
	if !sameCode(pending.UserCode, userCode) {
		return nil, fmt.Errorf("--user-code does not match the pending login for %s. Check the user code. If you started a newer login, complete it with its user code. %s", l.target(), startNewLoginHint)
	}
	if changed := l.changedSetting(*pending); changed != "" {
		return nil, fmt.Errorf("the login for %s was started with %s. Complete it with the same command that started it. %s", l.target(), changed, startNewLoginHint)
	}

	login, err := l.persistentAuth.NewDeviceCodeLogin(ctx)
	if err != nil {
		return nil, err
	}
	token, err := login.RequestDeviceAccessToken(ctx, pending.DeviceCode, confirmationCode)
	if err != nil {
		err = fmt.Errorf("token exchange: %w", err)
	}
	// The server no longer accepts a device code that it exchanged for a
	// token or considers invalid, so its pending login is deleted. A wrong
	// confirmation code leaves the device code valid, and a rate limit can be
	// lifted, so the pending login is kept for a rerun.
	switch {
	case err == nil:
		l.deletePendingLogin(ctx)
	case errors.Is(err, u2m.ErrInvalidConfirmationCode):
		err = fmt.Errorf("%w. Check the confirmation code and approve the login if you haven't, then rerun this command. %s if it still fails", err, startNewLoginHint)
		// The agent runs this step, so the error above is addressed to it.
		// Only the user may approve the login.
		if note := agent.AgentNote(ctx, "ask the user to check the confirmation code\n"+
			"and to approve the login if they haven't. Do not approve it yourself"); note != "" {
			err = fmt.Errorf("%w\n\n%s", err, note)
		}
		return nil, err
	case errors.Is(err, u2m.ErrDeviceCodeRateLimited):
		return nil, fmt.Errorf("%w. Rerun this command later. %s if it still fails", err, startNewLoginHint)
	case errors.Is(err, u2m.ErrInvalidDeviceCode):
		l.deletePendingLogin(ctx)
		return nil, fmt.Errorf("%w. %s", err, startNewLoginHint)
	}
	return token, err
}

// deletePendingLogin deletes the pending login. A pending login that is left
// over only fails a rerun of the second step, so an error is only logged.
func (l nonInteractiveDeviceCodeLogin) deletePendingLogin(ctx context.Context) {
	if err := l.tokenStore.Delete(l.key); err != nil {
		log.Debugf(ctx, "Failed to delete the pending device code login: %v", err)
	}
}

// changedSetting describes the setting that differs from the one pending was
// started with, or returns "" if none does.
func (l nonInteractiveDeviceCodeLogin) changedSetting(pending storage.PendingLogin) string {
	switch {
	case pending.Host != l.host:
		return "a different host"
	case pending.AccountID != l.accountID:
		return "a different account ID"
	case pending.ClientID != l.clientID:
		return "a different client ID"
	case !slices.Equal(pending.Scopes, l.scopes):
		return "different scopes"
	}
	return ""
}

// sameCode reports whether two device codes match when compared the way the
// user enters them.
func sameCode(a, b string) bool {
	return u2m.NormalizeCode(a) == u2m.NormalizeCode(b)
}

// promptForConfirmationCode shows the user where to approve the login and
// reads the confirmation code from stdin. If the server rejected the previous
// code, it shows the server's error and the number of attempts left instead
// and asks again.
func promptForConfirmationCode(ctx context.Context, da u2m.DeviceAuthorization, lastErr error, attemptsLeft int) (string, error) {
	if lastErr == nil {
		agentNote := deviceCodeAgentNote + "Ask the user for the confirmation code their browser shows, and enter it\n" +
			`at the "Confirmation code" prompt of this command.`
		cmdio.LogString(ctx, strings.Join(append(approvalInstructions(ctx, da, agentNote),
			"Once finished, enter the confirmation code provided in your browser.",
		), "\n"))
	} else {
		attempts := "attempts"
		if attemptsLeft == 1 {
			attempts = "attempt"
		}
		cmdio.LogString(ctx, fmt.Sprintf("%v. Enter the confirmation code again (%d %s left).", lastErr, attemptsLeft, attempts))
	}

	// A blank line is sent like any other code, so the server rejects it and
	// it counts as one of the attempts. A cancelled read fails the login, so
	// stdin is not read again after it.
	code, err := cmdio.AskCancellable(ctx, "Confirmation code", "")
	if err != nil {
		return "", fmt.Errorf("reading confirmation code: %w", err)
	}
	return code, nil
}

// approvalInstructions returns the lines that show the user where to approve
// the login. Both flows print them before saying how to complete the login.
// An AI agent also gets a note to pass the login on to the user, whether or
// not it runs the command in a terminal.
func approvalInstructions(ctx context.Context, da u2m.DeviceAuthorization, agentNote string) []string {
	lines := []string{
		"Using a browser on another device, visit:",
		da.VerificationURI,
		"",
		"And enter the code:",
		formatCode(da.UserCode),
		"",
	}
	if note := agent.AgentNote(ctx, agentNote); note != "" {
		lines = append(lines, note, "")
	}
	return lines
}

// formatCode displays an 8-character device code as two groups of four
// separated by a dash, matching how the browser displays codes.
func formatCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return code[:4] + "-" + code[4:]
}
