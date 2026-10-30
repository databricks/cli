package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// maxConfirmationAttempts is how many confirmation codes the user can enter
// for a single device code before the login fails.
const maxConfirmationAttempts = 3

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

// promptForConfirmationCode shows the user where to approve the login and
// reads the confirmation code from stdin. If the server rejected the previous
// code, it shows the server's error and the number of attempts left instead
// and asks again.
func promptForConfirmationCode(ctx context.Context, da u2m.DeviceAuthorization, lastErr error, attemptsLeft int) (string, error) {
	if lastErr == nil {
		cmdio.LogString(ctx, strings.Join([]string{
			"Using a browser on another device, visit:",
			da.VerificationURI,
			"",
			"And enter the code:",
			formatCode(da.UserCode),
			"",
			"Once finished, enter the confirmation code provided in your browser.",
		}, "\n"))
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

// formatCode displays an 8-character device code as two groups of four
// separated by a dash, matching how the browser displays codes.
func formatCode(code string) string {
	if len(code) != 8 {
		return code
	}
	return code[:4] + "-" + code[4:]
}
