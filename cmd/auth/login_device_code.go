package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
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

// deviceCodeIncompatibleFlags lists flags that device code login cannot honor.
var deviceCodeIncompatibleFlags = []string{
	// Resource indicators are sent on the browser authorization request,
	// which the device code flow does not make.
	"resource",
	// Device code login is only supported for workspace hosts.
	"skip-workspace",
}

var (
	errDeviceCodeNeedsHost     = errors.New("--use-device-code requires a workspace host. Pass it as an argument or with --host")
	errDeviceCodeWorkspaceOnly = errors.New("--use-device-code is only supported for workspace hosts")
	// The cluster picker is interactive.
	errDeviceCodeNonInteractiveCluster = errors.New("--configure-cluster is not supported by the non-interactive device code login. Set cluster_id on the profile after the login")
)

// isDeviceCodeSupportedForHost reports whether host is a workspace host. A
// unified host is only recognized by its account-scoped discoveryURL, so with
// an empty discoveryURL only classic account hosts are rejected.
func isDeviceCodeSupportedForHost(host, discoveryURL string) bool {
	return !isAccountOrUnifiedHost(host, discoveryURL)
}

// matchDeviceCodeProfiles matches the profiles that device code login does not
// always reject. Profiles on unified hosts still match, because the discovery
// URL that identifies them is not saved on the profile.
func matchDeviceCodeProfiles(p profile.Profile) bool {
	return isDeviceCodeSupportedForHost(p.Host, "") && len(u2mResourcesFromProfile(&p)) == 0
}

// validateDeviceCodeLogin returns an error if device code login is not enabled
// or is combined with flags it cannot honor.
func validateDeviceCodeLogin(ctx context.Context, cmd *cobra.Command) error {
	if enabled, _ := env.GetBool(ctx, deviceCodeEnvVar); !enabled {
		return fmt.Errorf("--use-device-code is experimental. Set %s=true to enable it", deviceCodeEnvVar)
	}
	for _, name := range deviceCodeIncompatibleFlags {
		// Compare with the default so that, e.g., --skip-workspace=false is
		// accepted.
		if f := cmd.Flag(name); f.Changed && f.Value.String() != f.DefValue {
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
// a non-interactive device code login are not set together.
func validateDeviceCodeCompletionFlags(useDeviceCode, hasUserCode, hasConfirmationCode bool) error {
	switch {
	case hasConfirmationCode && !hasUserCode:
		return errors.New("--confirmation-code requires --user-code")
	case hasUserCode && !hasConfirmationCode:
		return errors.New("--user-code requires --confirmation-code")
	case hasConfirmationCode && !useDeviceCode:
		return errors.New("--confirmation-code requires --use-device-code")
	}
	return nil
}

// nonInteractiveDeviceCodeLogin runs the two steps of a device code login
// that cannot read the confirmation code from stdin. The first step stores the
// device code as a pending login under key, and the second step exchanges it,
// so that the device code never leaves the machine.
type nonInteractiveDeviceCodeLogin struct {
	persistentAuth *u2m.PersistentAuth
	tokenStore     storage.Store
	key            string
	profile        string
	host           string
}

// pendingLoginKey returns the token store key of the pending login for the
// login with cacheKey.
func pendingLoginKey(cacheKey string) string {
	return cacheKey + pendingLoginKeySuffix
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
	login, err := l.persistentAuth.NewDeviceCodeLogin()
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
	}})
	if err != nil {
		return fmt.Errorf("store pending login: %w", err)
	}
	cmdio.LogString(ctx, strings.Join(append(approvalInstructions(*da),
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
	// Each profile or host has one pending login, so a newer login replaces
	// it. Checking the user code here avoids sending the confirmation code
	// with the wrong device code, which the server would reject as a wrong
	// confirmation code.
	if !sameCode(pending.UserCode, userCode) {
		return nil, fmt.Errorf("a newer login for %s replaced this one. Complete the newer login with its user code. %s", l.target(), startNewLoginHint)
	}

	login, err := l.persistentAuth.NewDeviceCodeLogin()
	if err != nil {
		return nil, err
	}
	token, err := login.RequestDeviceAccessToken(ctx, pending.DeviceCode, confirmationCode)
	if err != nil {
		err = fmt.Errorf("token exchange: %w", err)
	}
	// A wrong confirmation code leaves the device code valid, so the pending
	// login is kept for a rerun. Otherwise the server no longer accepts the
	// device code.
	if err == nil || errors.Is(err, u2m.ErrInvalidDeviceCode) {
		if err := l.tokenStore.Delete(l.key); err != nil {
			log.Debugf(ctx, "Failed to delete the pending device code login: %v", err)
		}
	}
	switch {
	case errors.Is(err, u2m.ErrInvalidConfirmationCode):
		return nil, fmt.Errorf("%w. Check the confirmation code and approve the login if you haven't, then rerun this command. %s if it still fails", err, startNewLoginHint)
	case errors.Is(err, u2m.ErrInvalidDeviceCode):
		return nil, fmt.Errorf("%w. %s", err, startNewLoginHint)
	}
	return token, err
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
		cmdio.LogString(ctx, strings.Join(append(approvalInstructions(da),
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
func approvalInstructions(da u2m.DeviceAuthorization) []string {
	return []string{
		"Using a browser on another device, visit:",
		da.VerificationURI,
		"",
		"And enter the code:",
		formatCode(da.UserCode),
		"",
	}
}

// formatCode displays an 8-character device code as two groups of four
// separated by a dash, matching how the browser displays codes.
func formatCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return code[:4] + "-" + code[4:]
}
