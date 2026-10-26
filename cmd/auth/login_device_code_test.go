package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
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

func TestValidateDeviceCodeHost(t *testing.T) {
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
		name    string
		args    auth.AuthArguments
		wantErr error
	}{
		{name: "workspace host", args: auth.AuthArguments{Host: workspace.URL}},
		{name: "workspace host with query params", args: auth.AuthArguments{Host: workspace.URL + "?o=12345"}},
		{name: "classic account host", args: auth.AuthArguments{Host: "https://accounts.cloud.databricks.test"}, wantErr: errDeviceCodeWorkspaceOnly},
		{name: "unified host", args: auth.AuthArguments{Host: unified.URL}, wantErr: errDeviceCodeWorkspaceOnly},
		{name: "unified host with query params", args: auth.AuthArguments{Host: unified.URL + "?o=12345"}, wantErr: errDeviceCodeWorkspaceOnly},
		{name: "unified host with account ID in query params", args: auth.AuthArguments{Host: unifiedWithoutAccountID.URL + "?a=acct-123"}, wantErr: errDeviceCodeWorkspaceOnly},
		{name: "unified host with account ID flag", args: auth.AuthArguments{Host: unifiedWithoutAccountID.URL, AccountID: "acct-123"}, wantErr: errDeviceCodeWorkspaceOnly},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			require.NoError(t, setHost(t.Context(), nil, &args, nil))
			assert.ErrorIs(t, validateDeviceCodeHost(&args), tt.wantErr)
		})
	}
}

// The test passes --profile instead of picking "Create a new profile" so that
// the host prompt is the only prompt. On Windows, bubbletea cannot cancel its
// read from the test's stdin pipe, so a finished prompt keeps reading and
// swallows the input typed for the next one.
func TestLoginDeviceCodeNewProfilePromptsForHost(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DATABRICKS_CONFIG_FILE", writeTempConfig(t, "[myprof]\nhost = https://workspace.test\n"))
	t.Setenv("DATABRICKS_AUTH_STORAGE", "plaintext")
	t.Setenv(deviceCodeEnvVar, "true")

	tm := termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		cmd := newLoginCommand(&auth.AuthArguments{})
		cmd.Flags().String("profile", "", "")
		cmd.SetContext(ctx)
		cmd.SetArgs([]string{"--use-device-code", "--profile", "newprof"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		return struct{}{}, cmd.Execute()
	})
	// Without a host, device code login asks for one instead of starting the
	// discovery login.
	tm.WaitFor("Databricks host")
	tm.Type(termtest.KeyCtrlC)
	_, err := tm.Result()
	assert.EqualError(t, err, "^C")
}
