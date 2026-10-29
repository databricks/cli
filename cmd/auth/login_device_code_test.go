package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
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

// setUpDeviceCodeLogin isolates the environment of a device code login, using
// config as the contents of .databrickscfg.
func setUpDeviceCodeLogin(t *testing.T, config string) {
	testutil.CleanupEnvironment(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DATABRICKS_CONFIG_FILE", writeTempConfig(t, config))
	t.Setenv("DATABRICKS_AUTH_STORAGE", "plaintext")
	t.Setenv(deviceCodeEnvVar, "true")
}

// runDeviceCodeLogin runs "auth login --use-device-code" with args.
func runDeviceCodeLogin(ctx context.Context, args ...string) error {
	authArguments := &auth.AuthArguments{}
	cmd := newLoginCommand(authArguments)
	cmd.Flags().StringVar(&authArguments.Host, "host", "", "")
	cmd.Flags().String("profile", "", "")
	cmd.SetContext(ctx)
	cmd.SetArgs(append([]string{"--use-device-code"}, args...))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}

// startDeviceCodeLogin runs "auth login --use-device-code" with args in a
// terminal emulator, using config as the contents of .databrickscfg.
func startDeviceCodeLogin(t *testing.T, config string, args ...string) *termtest.Term[struct{}] {
	setUpDeviceCodeLogin(t, config)
	return termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, runDeviceCodeLogin(ctx, args...)
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

func TestValidateDeviceCodeCompletionFlags(t *testing.T) {
	tests := []struct {
		name          string
		useDeviceCode bool
		flags         map[string]string
		wantErr       string
	}{
		{name: "neither", useDeviceCode: true},
		{name: "both", useDeviceCode: true, flags: map[string]string{"user-code": "WDJB-MJHT", "confirmation-code": "FTFA-QQKY"}},
		{name: "confirmation code only", useDeviceCode: true, flags: map[string]string{"confirmation-code": "FTFA-QQKY"}, wantErr: "--confirmation-code requires --user-code"},
		{name: "user code only", useDeviceCode: true, flags: map[string]string{"user-code": "WDJB-MJHT"}, wantErr: "--user-code requires --confirmation-code"},
		{name: "without device code", flags: map[string]string{"user-code": "WDJB-MJHT", "confirmation-code": "FTFA-QQKY"}, wantErr: "--confirmation-code requires --use-device-code"},
		{name: "empty user code", useDeviceCode: true, flags: map[string]string{"user-code": "", "confirmation-code": "FTFA-QQKY"}, wantErr: "--user-code cannot be empty"},
		{name: "blank confirmation code", useDeviceCode: true, flags: map[string]string{"user-code": "WDJB-MJHT", "confirmation-code": "  "}, wantErr: "--confirmation-code cannot be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("user-code", "", "")
			cmd.Flags().String("confirmation-code", "", "")
			for name, value := range tt.flags {
				require.NoError(t, cmd.Flags().Set(name, value))
			}

			err := validateDeviceCodeCompletionFlags(cmd, tt.useDeviceCode)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSameCode(t *testing.T) {
	assert.True(t, sameCode("WDJBMJHT", "WDJB-MJHT"))
	assert.True(t, sameCode("WDJBMJHT", "wdjbmjht"))
	assert.True(t, sameCode("WDJBMJHT", " WDJB-MJHT\n"))
	assert.False(t, sameCode("WDJBMJHT", "WDJB-MJHA"))
}

// newDeviceCodeServer returns a fake workspace with device code login.
func newDeviceCodeServer(t *testing.T) *testserver.Server {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	return server
}

// A wrong or blank confirmation code leaves the device code valid, so the user
// can enter it again. The code is accepted the way the browser displays it, in
// any case.
func TestLoginDeviceCodeReadsConfirmationCode(t *testing.T) {
	server := newDeviceCodeServer(t)
	tm := startDeviceCodeLogin(t, "[test]\nhost = "+server.URL+"\n", "--profile", "test")
	tm.WaitFor("Confirmation code")
	tm.Type("AAAA-AAAA\n")
	tm.WaitFor("Enter the confirmation code again (2 attempts left)")
	tm.Type("\n")
	tm.WaitFor("Enter the confirmation code again (1 attempt left)")
	tm.Type("ftfa-qqky\n")
	_, err := tm.Result()
	require.NoError(t, err)

	p, err := loadProfileByName(t.Context(), "test", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, server.URL, p.Host)
	assert.Equal(t, strconv.Itoa(testserver.TestWorkspaceID), p.WorkspaceID)
	assert.Equal(t, authTypeDatabricksCLI, p.AuthType)

	raw, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".databricks", "token-cache.json"))
	require.NoError(t, err)
	var cache struct {
		Tokens map[string]json.RawMessage `json:"tokens"`
	}
	require.NoError(t, json.Unmarshal(raw, &cache))
	assert.Contains(t, cache.Tokens, "test")
}

// The token gets the scopes requested at device authorization, so --scopes is
// sent there and not with the token request.
func TestLoginDeviceCodeRequestsScopes(t *testing.T) {
	server := newDeviceCodeServer(t)
	var scope string
	server.RequestCallback = func(r *testserver.Request) {
		if r.URL.Path == "/oidc/v1/device/auth" {
			form, err := url.ParseQuery(string(r.Body))
			assert.NoError(t, err)
			scope = form.Get("scope")
		}
	}
	tm := startDeviceCodeLogin(t, "", "--host", server.URL, "--profile", "test", "--scopes", "sql")
	tm.WaitFor("Confirmation code")
	tm.Type("FTFA-QQKY\n")
	_, err := tm.Result()
	require.NoError(t, err)
	assert.Equal(t, "offline_access sql", scope)

	p, err := loadProfileByName(t.Context(), "test", profile.DefaultProfiler)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, server.URL, p.Host)
	assert.Equal(t, strconv.Itoa(testserver.TestWorkspaceID), p.WorkspaceID)
	assert.Equal(t, "sql", p.Scopes)
}

// An agent without a terminal can start the login, and the user can complete
// it in a terminal. The second step does not prompt for a profile, so it finds
// the pending login under the host, like the first step did.
func TestLoginDeviceCodeCompletesInTerminal(t *testing.T) {
	server := newDeviceCodeServer(t)
	setUpDeviceCodeLogin(t, "")
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, "", ""))
	require.NoError(t, runDeviceCodeLogin(ctx, "--host", server.URL))

	tm := termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, runDeviceCodeLogin(ctx, "--host", server.URL, "--user-code", "WDJB-MJHT", "--confirmation-code", "FTFA-QQKY")
	})
	_, err := tm.Result()
	require.NoError(t, err)

	raw, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".databricks", "token-cache.json"))
	require.NoError(t, err)
	var cache struct {
		Tokens        map[string]json.RawMessage `json:"tokens"`
		PendingLogins map[string]json.RawMessage `json:"pending_logins"`
	}
	require.NoError(t, json.Unmarshal(raw, &cache))
	assert.Contains(t, cache.Tokens, server.URL)
	assert.Empty(t, cache.PendingLogins)

	profiles, err := profile.DefaultProfiler.LoadProfiles(t.Context(), profile.MatchAllProfiles)
	require.NoError(t, err)
	assert.Empty(t, profiles)
}

// A user code that does not match the pending login fails before the token
// exchange, so the confirmation code is not sent with the device code of
// another login.
func TestLoginDeviceCodeCompletionChecksUserCode(t *testing.T) {
	server := newDeviceCodeServer(t)
	var tokenRequests atomic.Int32
	server.RequestCallback = func(req *testserver.Request) {
		if req.URL.Path == "/oidc/v1/token" {
			tokenRequests.Add(1)
		}
	}
	setUpDeviceCodeLogin(t, "")
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, "", ""))
	require.NoError(t, runDeviceCodeLogin(ctx, "--host", server.URL, "--profile", "test"))

	err := runDeviceCodeLogin(ctx, "--host", server.URL, "--profile", "test", "--user-code", "AAAA-AAAA", "--confirmation-code", "FTFA-QQKY")
	assert.EqualError(t, err, "a newer login for profile test replaced this one. Complete the newer login with its user code. "+startNewLoginHint)
	assert.Zero(t, tokenRequests.Load())

	// The pending login is kept, and the user code is compared the way the
	// user enters it.
	require.NoError(t, runDeviceCodeLogin(ctx, "--host", server.URL, "--profile", "test", "--user-code", "wdjbmjht", "--confirmation-code", "FTFA-QQKY"))
	assert.Equal(t, int32(1), tokenRequests.Load())
}

func TestLoginDeviceCodeCompletionChecksSettings(t *testing.T) {
	tests := []struct {
		name string
		// startArgs and completeArgs are added to the flags of each step.
		startArgs    []string
		completeArgs []string
		// completeOnOtherHost completes the login on a second server.
		completeOnOtherHost bool
		wantChanged         string
	}{
		{
			name:                "host",
			completeOnOtherHost: true,
			wantChanged:         "a different host",
		},
		{
			name:         "client ID",
			completeArgs: []string{"--client-id", "other-client"},
			wantChanged:  "a different client ID",
		},
		{
			name:        "scopes dropped",
			startArgs:   []string{"--scopes", "sql"},
			wantChanged: "different scopes",
		},
		{
			name:         "scopes added",
			completeArgs: []string{"--scopes", "sql"},
			wantChanged:  "different scopes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newDeviceCodeServer(t)
			var tokenRequests atomic.Int32
			server.RequestCallback = func(req *testserver.Request) {
				if req.URL.Path == "/oidc/v1/token" {
					tokenRequests.Add(1)
				}
			}
			setUpDeviceCodeLogin(t, "")
			ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, "", ""))
			require.NoError(t, runDeviceCodeLogin(ctx, append([]string{"--host", server.URL, "--profile", "test"}, tt.startArgs...)...))

			host := server.URL
			if tt.completeOnOtherHost {
				other := newDeviceCodeServer(t)
				other.RequestCallback = server.RequestCallback
				host = other.URL
			}
			args := append([]string{"--host", host, "--profile", "test", "--user-code", "WDJB-MJHT", "--confirmation-code", "FTFA-QQKY"}, tt.completeArgs...)
			err := runDeviceCodeLogin(ctx, args...)
			assert.EqualError(t, err, "the login for profile test was started with "+tt.wantChanged+". Complete it with the same command that started it. "+startNewLoginHint)
			assert.Zero(t, tokenRequests.Load())
		})
	}
}

func TestLoginDeviceCodeCompletionWithoutHostSkipsPicker(t *testing.T) {
	tm := startDeviceCodeLogin(t, "", "--user-code", "WDJB-MJHT", "--confirmation-code", "FTFA-QQKY")
	_, err := tm.Result()
	assert.ErrorIs(t, err, errDeviceCodeNeedsHost)
}

// Only a wrong confirmation code can be fixed by entering the code again, so
// any other token exchange error ends the login.
func TestLoginDeviceCodeFailsOnTokenExchangeError(t *testing.T) {
	// Neither error can be fixed by entering the confirmation code again, so
	// the login fails after one token request instead of asking again.
	tests := []struct {
		name        string
		status      int
		description string
		wantErr     string
	}{
		{
			name:        "rate limited",
			status:      http.StatusTooManyRequests,
			description: "Device grant rate limit exceeded",
			wantErr:     "token exchange: Device grant rate limit exceeded (error code: invalid_request)",
		},
		{
			name:        "unknown device code",
			status:      http.StatusBadRequest,
			description: "Invalid device code",
			wantErr:     "token exchange: Invalid device code (error code: invalid_request)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			var tokenRequests atomic.Int32
			// The router keeps the first handler registered for a route, so
			// this one replaces the default token endpoint.
			server.Handle("POST", "/oidc/v1/token", func(_ testserver.Request) any {
				tokenRequests.Add(1)
				return testserver.Response{
					StatusCode: tt.status,
					Body:       map[string]string{"error": "invalid_request", "error_description": tt.description},
				}
			})
			testserver.AddDefaultHandlers(server)
			tm := startDeviceCodeLogin(t, "[test]\nhost = "+server.URL+"\n", "--profile", "test")
			tm.WaitFor("Confirmation code")
			tm.Type("FTFA-QQKY\n")
			_, err := tm.Result()
			assert.EqualError(t, err, tt.wantErr)
			assert.Equal(t, int32(1), tokenRequests.Load())
		})
	}
}

func TestLoginDeviceCodeWrongConfirmationCode(t *testing.T) {
	// The prompt for each attempt, which the test waits for before typing
	// the next code.
	prompts := []string{"Confirmation code", "(2 attempts left)", "(1 attempt left)"}
	tests := []struct {
		name  string
		codes []string
		// closeStdin ends the input after the codes, so that the next
		// prompt fails.
		closeStdin        bool
		wantErr           string
		wantTokenRequests int32
	}{
		{
			name:              "right code after a wrong one",
			codes:             []string{"AAAA-AAAA", "FTFA-QQKY"},
			wantTokenRequests: 2,
		},
		{
			name:              "too many wrong codes",
			codes:             []string{"AAAA-AAAA", "BBBB-BBBB", "CCCC-CCCC"},
			wantErr:           "Invalid confirmation code (error code: invalid_request). Start login from scratch",
			wantTokenRequests: 3,
		},
		{
			name:              "no code after a wrong one",
			codes:             []string{"AAAA-AAAA"},
			closeStdin:        true,
			wantErr:           "reading confirmation code: EOF",
			wantTokenRequests: 1,
		},
		{
			name:              "no code",
			closeStdin:        true,
			wantErr:           "reading confirmation code: EOF",
			wantTokenRequests: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			testserver.AddDefaultHandlers(server)
			var tokenRequests atomic.Int32
			server.RequestCallback = func(req *testserver.Request) {
				if req.URL.Path == "/oidc/v1/token" {
					tokenRequests.Add(1)
				}
			}
			tm := startDeviceCodeLogin(t, "[test]\nhost = "+server.URL+"\n", "--profile", "test")
			for i, code := range tt.codes {
				tm.WaitFor(prompts[i])
				tm.Type(code + "\n")
			}
			if tt.closeStdin {
				tm.WaitFor(prompts[len(tt.codes)])
				tm.Close()
			}
			_, err := tm.Result()
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantTokenRequests, tokenRequests.Load())
		})
	}
}
