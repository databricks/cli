package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/testserver"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDeviceCodeLogin(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		setFlag string
		flagVal string
		wantErr string
	}{
		{
			name:    "disabled by default",
			wantErr: "--use-device-code is experimental. Set DATABRICKS_CLI_DEVICE_CODE=true to enable it",
		},
		{
			name:    "explicitly disabled",
			envVal:  "false",
			wantErr: "--use-device-code is experimental. Set DATABRICKS_CLI_DEVICE_CODE=true to enable it",
		},
		{
			name:   "enabled",
			envVal: "true",
		},
		{
			name:    "resource is incompatible",
			envVal:  "1",
			setFlag: "resource",
			flagVal: "https://workspace.test/ai-gateway/mcp-services/system.ai.github",
			wantErr: "--use-device-code cannot be combined with --resource",
		},
		{
			name:    "skip-workspace is incompatible",
			envVal:  "1",
			setFlag: "skip-workspace",
			flagVal: "true",
			wantErr: "--use-device-code cannot be combined with --skip-workspace",
		},
		{
			name:    "skip-workspace set to its default",
			envVal:  "1",
			setFlag: "skip-workspace",
			flagVal: "false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), deviceCodeEnvVar, tt.envVal)
			cmd := &cobra.Command{}
			cmd.Flags().StringArray("resource", nil, "")
			cmd.Flags().Bool("skip-workspace", false, "")
			if tt.setFlag != "" {
				require.NoError(t, cmd.Flags().Set(tt.setFlag, tt.flagVal))
			}

			err := validateDeviceCodeLogin(ctx, cmd)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPromptForConfirmationCode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "code", input: "FTFA-QQKY\n", want: "FTFA-QQKY"},
		{name: "code without newline", input: "FTFA-QQKY", want: "FTFA-QQKY"},
		{name: "blank line", input: "  \nFTFA-QQKY\n", want: "  "},
		{name: "no input", input: "", wantErr: "reading confirmation code: EOF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(tt.input), &bytes.Buffer{}, &stderr, "", ""))

			code, err := promptForConfirmationCode(ctx, u2m.DeviceAuthorization{
				UserCode:        "WDJBMJHT",
				VerificationURI: "https://workspace.test/device/verification",
			}, nil, 3)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, code)
			}
			assert.True(t, strings.HasPrefix(stderr.String(), strings.Join([]string{
				"Using a browser on another device, visit:",
				"https://workspace.test/device/verification",
				"",
				"And enter the code:",
				"WDJB-MJHT",
				"",
				"Once finished, enter the confirmation code provided in your browser.",
				"Confirmation code: ",
			}, "\n")), stderr.String())
		})
	}
}

func TestPromptForConfirmationCode_Retry(t *testing.T) {
	tests := []struct {
		attemptsLeft int
		want         string
	}{
		{attemptsLeft: 2, want: "wrong code. Enter the confirmation code again (2 attempts left).\nConfirmation code: "},
		{attemptsLeft: 1, want: "wrong code. Enter the confirmation code again (1 attempt left).\nConfirmation code: "},
	}
	for _, tt := range tests {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader("FTFA-QQKY\n"), &bytes.Buffer{}, &stderr, "", ""))

		code, err := promptForConfirmationCode(ctx, u2m.DeviceAuthorization{
			UserCode:        "WDJBMJHT",
			VerificationURI: "https://workspace.test/device/verification",
		}, errors.New("wrong code"), tt.attemptsLeft)
		require.NoError(t, err)
		assert.Equal(t, "FTFA-QQKY", code)
		// The instructions are only shown before the first attempt.
		assert.Equal(t, tt.want, stderr.String())
	}
}

func TestPromptForConfirmationCode_Timeout(t *testing.T) {
	// The pipe is never written to, so reading the code blocks until the
	// context is done.
	stdin, _ := io.Pipe()
	defer stdin.Close()
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, stdin, &bytes.Buffer{}, &bytes.Buffer{}, "", ""))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	_, err := promptForConfirmationCode(ctx, u2m.DeviceAuthorization{
		UserCode:        "WDJBMJHT",
		VerificationURI: "https://workspace.test/device/verification",
	}, nil, 3)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.EqualError(t, err, "reading confirmation code: context deadline exceeded")
}

func TestFormatCode(t *testing.T) {
	assert.Equal(t, "WDJB-MJHT", formatCode("WDJBMJHT"))
	assert.Equal(t, "ABC", formatCode("ABC"))
}

func TestIsDeviceCodeSupportedForHost(t *testing.T) {
	workspace := newDiscoveryServer(t, map[string]any{
		"account_id":    "acct-123",
		"workspace_id":  "12345",
		"oidc_endpoint": "https://workspace.test/oidc",
	})
	unified := newDiscoveryServer(t, map[string]any{
		"account_id":    "acct-123",
		"oidc_endpoint": "https://spog.test/oidc/accounts/acct-123",
	})
	// Without an account ID, host discovery cannot fill in the placeholder
	// and does not resolve the account-scoped discovery URL.
	unifiedWithoutAccountID := newDiscoveryServer(t, map[string]any{
		"oidc_endpoint": "https://spog.test/oidc/accounts/{account_id}",
	})

	tests := []struct {
		name string
		args auth.AuthArguments
		want bool
	}{
		{name: "workspace host", args: auth.AuthArguments{Host: workspace.URL}, want: true},
		{name: "workspace host with query params", args: auth.AuthArguments{Host: workspace.URL + "?o=12345"}, want: true},
		{name: "classic account host", args: auth.AuthArguments{Host: "https://accounts.cloud.databricks.test"}},
		{name: "unified host", args: auth.AuthArguments{Host: unified.URL}},
		{name: "unified host with query params", args: auth.AuthArguments{Host: unified.URL + "?o=12345"}},
		{name: "unified host with account ID in query params", args: auth.AuthArguments{Host: unifiedWithoutAccountID.URL + "?a=acct-123"}},
		{name: "unified host with account ID flag", args: auth.AuthArguments{Host: unifiedWithoutAccountID.URL, AccountID: "acct-123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			require.NoError(t, setHost(t.Context(), nil, &args, nil))
			assert.Equal(t, tt.want, isDeviceCodeSupportedForHost(args.Host, args.DiscoveryURL))
		})
	}
}

// startDeviceCodeLogin runs "auth login --use-device-code" with args in a
// terminal emulator, using config as the contents of .databrickscfg.
func startDeviceCodeLogin(t *testing.T, config string, args ...string) *termtest.Term[struct{}] {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DATABRICKS_CONFIG_FILE", writeTempConfig(t, config))
	t.Setenv("DATABRICKS_AUTH_STORAGE", "plaintext")
	t.Setenv(deviceCodeEnvVar, "true")

	return termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		cmd := newLoginCommand(&auth.AuthArguments{})
		cmd.Flags().String("profile", "", "")
		cmd.SetContext(ctx)
		cmd.SetArgs(append([]string{"--use-device-code"}, args...))
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		return struct{}{}, cmd.Execute()
	})
}

func TestLoginDeviceCodeNewProfilePromptsForHost(t *testing.T) {
	tm := startDeviceCodeLogin(t, "", "--profile", "newprof")
	tm.WaitFor("Databricks host")
	tm.Type(termtest.KeyEnter)
	_, err := tm.Result()
	assert.ErrorIs(t, err, errDeviceCodeNeedsHost)
}

func TestLoginDeviceCodePickerEmptyHost(t *testing.T) {
	// termtest feeds stdin through a pipe. On Windows, bubbletea cannot
	// cancel its read on a pipe, so the picker's reader is still blocked
	// after it exits and takes the Enter meant for the host prompt.
	if runtime.GOOS == "windows" {
		t.Skip("bubbletea cannot cancel reads on non-console input on Windows")
	}
	tm := startDeviceCodeLogin(t, "")
	tm.WaitFor(profilePickerEnterHostLabel)
	tm.Type(termtest.KeyDown + termtest.KeyEnter)
	tm.WaitFor("Databricks host")
	tm.Type(termtest.KeyEnter)
	_, err := tm.Result()
	assert.ErrorIs(t, err, errDeviceCodeNeedsHost)
}

func TestMatchDeviceCodeProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile profile.Profile
		want    bool
	}{
		{name: "workspace host", profile: profile.Profile{Host: "https://workspace.test"}, want: true},
		// setHost prompts for the host of a profile without one.
		{name: "no host", profile: profile.Profile{}, want: true},
		{name: "classic account host", profile: profile.Profile{Host: "https://accounts.cloud.databricks.test", AccountID: "acct-123"}},
		{name: "classic account host for DoD", profile: profile.Profile{Host: "https://accounts-dod.cloud.databricks.test", AccountID: "acct-123"}},
		{
			name:    "scoped to resources",
			profile: profile.Profile{Host: "https://workspace.test", AuthType: authTypeDatabricksCLI, Resources: "https://workspace.test/ai-gateway/mcp-services/system.ai.github"},
		},
		// Login only reuses the resources of a databricks-cli profile.
		{
			name:    "resources on another auth type",
			profile: profile.Profile{Host: "https://workspace.test", AuthType: "pat", Resources: "https://workspace.test/ai-gateway/mcp-services/system.ai.github"},
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchDeviceCodeProfiles(tt.profile))
		})
	}
}

func TestLoginDeviceCodePickerHidesUnsupportedProfiles(t *testing.T) {
	tm := startDeviceCodeLogin(t, `[workspace-prof]
host = https://workspace.test

[account-prof]
host = https://accounts.cloud.databricks.test
account_id = acct-123

[scoped-prof]
host = https://workspace.test
auth_type = databricks-cli
resources = https://workspace.test/ai-gateway/mcp-services/system.ai.github
`)
	tm.WaitFor(profilePickerEnterHostLabel)
	screen := tm.Snapshot()
	assert.Contains(t, screen, "workspace-prof")
	assert.NotContains(t, screen, "account-prof")
	assert.NotContains(t, screen, "scoped-prof")
	tm.Type(termtest.KeyCtrlC)
	_, err := tm.Result()
	assert.EqualError(t, err, "^C")
}

// Only a wrong confirmation code can be fixed by entering the code again, so
// any other token exchange error ends the login.
func TestLoginDeviceCodeFailsOnTokenExchangeError(t *testing.T) {
	server := testserver.New(t)
	// The router keeps the first handler registered for a route, so this one
	// replaces the default token endpoint.
	server.Handle("POST", "/oidc/v1/token", func(_ testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       map[string]string{"error": "invalid_request", "error_description": "Device grant rate limit exceeded"},
		}
	})
	testserver.AddDefaultHandlers(server)
	tm := startDeviceCodeLogin(t, "[test]\nhost = "+server.URL+"\n", "--profile", "test")
	tm.WaitFor("Confirmation code")
	tm.Type("FTFA-QQKY\n")
	_, err := tm.Result()
	assert.EqualError(t, err, "token exchange: Device grant rate limit exceeded (error code: invalid_request)")
}
