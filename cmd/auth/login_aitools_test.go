package auth

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/testserver"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// askedMarker is what stubAskAtLogin prints when the login asks.
const askedMarker = "<asked about Databricks AI tools>"

// stubAskAtLogin replaces autoinstall.AskAtLogin, which main.go sets and
// cmd/aitools implements, with one that prints askedMarker, and returns the
// number of calls. The variable is global, so tests that call it must not run
// in parallel.
func stubAskAtLogin(t *testing.T) *int {
	calls := 0
	prev := autoinstall.AskAtLogin
	t.Cleanup(func() { autoinstall.AskAtLogin = prev })
	autoinstall.AskAtLogin = func(cmd *cobra.Command) {
		calls++
		cmdio.LogString(cmd.Context(), askedMarker)
	}
	return &calls
}

// assertAskedAfterSaving checks that the login asked once, after it printed
// that it saved profile.
func assertAskedAfterSaving(t *testing.T, output string, calls int, profile string) {
	t.Helper()
	assert.Equal(t, 1, calls)
	saved := strings.Index(output, "Profile "+profile+" was successfully saved")
	asked := strings.Index(output, askedMarker)
	require.NotEqual(t, -1, saved, "the profile was saved; output:\n%s", output)
	assert.Greater(t, asked, saved, "asked after the profile was saved; output:\n%s", output)
}

// skipAfterAPromptOnWindows skips a test that answers a prompt and then types
// more: see TestLoginDeviceCodePickerEmptyHost.
func skipAfterAPromptOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bubbletea cannot cancel reads on non-console input on Windows")
	}
}

// A login that names no profile asks after it saved the profile. The device
// code login runs the same steps after its token as the browser login.
func TestLoginAsksAboutAIToolsAfterSavingTheProfile(t *testing.T) {
	skipAfterAPromptOnWindows(t)
	calls := stubAskAtLogin(t)
	server := newDeviceCodeServer(t)
	tm := startDeviceCodeLogin(t, "", "--host", server.URL)
	tm.WaitFor("Databricks profile name")
	tm.Type("newprof" + termtest.KeyEnter)
	tm.WaitFor("Confirmation code")
	tm.Type("FTFA-QQKY\n")
	_, err := tm.Result()
	require.NoError(t, err)
	assertAskedAfterSaving(t, tm.Raw(), *calls, "newprof")
}

// The login through login.databricks.com saves the profile in discoveryLogin,
// and asks after it too.
func TestLoginAsksAboutAIToolsAfterTheDiscoveryLogin(t *testing.T) {
	skipAfterAPromptOnWindows(t)
	calls := stubAskAtLogin(t)
	server := newDeviceCodeServer(t)
	server.Handle("GET", "/api/2.0/tokens/introspect", func(testserver.Request) any {
		return map[string]any{}
	})
	setUpLogin(t, "")
	// The login prints the URL instead of opening a browser.
	t.Setenv("BROWSER", "none")
	tm := termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		authArguments := &auth.AuthArguments{}
		cmd := newLoginCommand(authArguments)
		cmd.Flags().StringVar(&authArguments.Host, "host", "", "")
		cmd.Flags().String("profile", "", "")
		// The discovery login checks that these flags of `databricks auth`
		// are not set.
		cmd.Flags().StringVar(&authArguments.AccountID, "account-id", "", "")
		cmd.Flags().StringVar(&authArguments.WorkspaceID, "workspace-id", "", "")
		cmd.SetContext(ctx)
		cmd.SetArgs([]string{})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		return struct{}{}, cmd.Execute()
	})
	tm.WaitFor(profilePickerCreateNewLabel)
	tm.Type(termtest.KeyEnter)
	tm.WaitFor("Databricks profile name")
	tm.Type("newprof" + termtest.KeyEnter)
	output := tm.WaitFor("opening this link in your browser:")
	completeDiscoveryLogin(t, tm, output, server.URL+"/oidc")
	_, err := tm.Result()
	require.NoError(t, err)
	assertAskedAfterSaving(t, tm.Raw(), *calls, "newprof")
}

// completeDiscoveryLogin does what the browser does at login.databricks.com:
// it sends the OAuth callback of the URL that the login printed, with issuer
// as the workspace the user picked. See acceptance/bin/discovery_browser.py.
func completeDiscoveryLogin(t *testing.T, tm *termtest.Term[struct{}], output, issuer string) {
	t.Helper()
	link := regexp.MustCompile(`https?://\S+`)
	var loginURL string
	for loginURL == "" {
		_, after, _ := strings.Cut(output, "opening this link in your browser:")
		loginURL = link.FindString(after)
		if loginURL == "" {
			output = tm.WaitFor("destination_url")
		}
	}
	u, err := url.Parse(loginURL)
	require.NoError(t, err)
	dest, err := url.Parse(u.Query().Get("destination_url"))
	require.NoError(t, err)
	callback, err := url.Parse(dest.Query().Get("redirect_uri"))
	require.NoError(t, err)
	callback.RawQuery = url.Values{
		"code":  {"oauth-code"},
		"state": {dest.Query().Get("state")},
		"iss":   {issuer},
	}.Encode()
	resp, err := http.Get(callback.String())
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// A login that names its profile, with --profile or with a positional
// argument that resolves to one, does not ask. A positional host is not a
// profile.
func TestLoginAsksAboutAIToolsOnlyWithoutANamedProfile(t *testing.T) {
	server := newDeviceCodeServer(t)
	config := "[test]\nhost = " + server.URL + "\n"
	tests := []struct {
		name    string
		args    []string
		profile string
		// promptsForName is set when the login asks for the profile name,
		// which the test answers with profile.
		promptsForName bool
		asks           bool
	}{
		{name: "--profile", args: []string{"--profile", "test"}, profile: "test"},
		{name: "--profile of a new profile", args: []string{"--host", server.URL, "--profile", "newprof"}, profile: "newprof"},
		{name: "positional profile", args: []string{"test"}, profile: "test"},
		{name: "positional host", args: []string{server.URL}, profile: "newprof", promptsForName: true, asks: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.promptsForName {
				skipAfterAPromptOnWindows(t)
			}
			calls := stubAskAtLogin(t)
			tm := startDeviceCodeLogin(t, config, tc.args...)
			if tc.promptsForName {
				tm.WaitFor("Databricks profile name")
				tm.Type("newprof" + termtest.KeyEnter)
			}
			tm.WaitFor("Confirmation code")
			tm.Type("FTFA-QQKY\n")
			_, err := tm.Result()
			require.NoError(t, err)
			output := tm.Raw()
			assert.Contains(t, output, "Profile "+tc.profile+" was successfully saved")
			if tc.asks {
				assertAskedAfterSaving(t, output, *calls, tc.profile)
			} else {
				assert.Zero(t, *calls)
				assert.NotContains(t, output, askedMarker)
			}
		})
	}
}

func TestLoginAsksAboutAITools(t *testing.T) {
	assert.True(t, loginAsksAboutAITools("", false))
	assert.False(t, loginAsksAboutAITools("test", false), "a named profile")
	assert.False(t, loginAsksAboutAITools("", true), "the non-interactive device code login")
}

// The device code login that an agent without a terminal starts stores a
// pending login and saves no profile; its second step, which completes it,
// does not ask either.
func TestLoginDeviceCodeWithoutATerminalDoesNotAsk(t *testing.T) {
	calls := stubAskAtLogin(t)
	server := newDeviceCodeServer(t)
	setUpLogin(t, "")
	ctx := cmdio.MockDiscard(t.Context())
	require.NoError(t, runDeviceCodeLogin(ctx, "--host", server.URL))
	tm := termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, runDeviceCodeLogin(ctx, "--host", server.URL, "--user-code", "WDJB-0001", "--confirmation-code", "FTFA-QQKY")
	})
	_, err := tm.Result()
	require.NoError(t, err)
	assert.Zero(t, *calls)
}

// Without the function that main.go sets, as in tests, the login asks nothing.
func TestLoginWithoutAskAtLogin(t *testing.T) {
	skipAfterAPromptOnWindows(t)
	prev := autoinstall.AskAtLogin
	t.Cleanup(func() { autoinstall.AskAtLogin = prev })
	autoinstall.AskAtLogin = nil
	server := newDeviceCodeServer(t)
	tm := startDeviceCodeLogin(t, "", "--host", server.URL)
	tm.WaitFor("Databricks profile name")
	tm.Type("newprof" + termtest.KeyEnter)
	tm.WaitFor("Confirmation code")
	tm.Type("FTFA-QQKY\n")
	_, err := tm.Result()
	require.NoError(t, err)
	assert.Contains(t, tm.Raw(), "Profile newprof was successfully saved")
}
