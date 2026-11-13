package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	loginTestService     = "system.ai.github"
	loginTestCredPath    = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	loginTestServicePath = "/api/2.1/unity-catalog/mcp-services/system.ai.github"
)

// loginTestServer answers the credential endpoints from a script: each GET returns the next
// entry of states, where "" means NOT_FOUND, and the last entry repeats. DELETE answers
// deleted (204) or not found (404).
type loginTestServer struct {
	mu      sync.Mutex
	states  []string
	reads   int
	deleted bool
	// signedIn is what DELETE answers: true for 200, false for 404.
	signedIn bool
	// gate, when set, holds the credential at NOT_FOUND until it returns true. The script does not advance meanwhile.
	gate func() bool
	// gets counts the credential GETs, and fail, when set, answers GET number get (from 1) with the status it returns,
	// or 0 to answer from the script.
	gets int
	fail func(get int) int
}

func (s *loginTestServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rw.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == loginTestServicePath && r.Method == http.MethodGet:
		_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","config":{"source_connection":{"options":{"oauth_provider":"OAUTH_PROVIDER_GITHUB_MCP"}}}}`)
	case r.URL.Path == loginTestCredPath && r.Method == http.MethodGet:
		s.gets++
		if s.fail != nil {
			if status := s.fail(s.gets); status != 0 {
				http.Error(rw, `{"error_code":"TEMPORARILY_UNAVAILABLE","message":"try again"}`, status)
				return
			}
		}
		state := ""
		if s.gate == nil || s.gate() {
			state = s.states[min(s.reads, len(s.states)-1)]
			s.reads++
		}
		if state == "" {
			http.Error(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`, http.StatusNotFound)
			return
		}
		body, _ := json.Marshal(catalog.McpServiceUserMappedCredential{
			ProvisioningInfo: &catalog.ProvisioningInfo{State: catalog.ProvisioningInfoState(state)},
		})
		_, _ = rw.Write(body)
	case r.URL.Path == loginTestCredPath && r.Method == http.MethodDelete:
		if !s.signedIn {
			http.Error(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`, http.StatusNotFound)
			return
		}
		s.deleted = true
		_, _ = io.WriteString(rw, `{}`)
	default:
		// The SDK resolves host metadata with requests of its own; those are not the endpoints under test.
		http.NotFound(rw, r)
	}
}

// loginTestSignInURL is the sign-in URL that SignIn is given for the fake's workspace, with workspace ID 123.
func loginTestSignInURL(w *databricks.WorkspaceClient) string {
	return SignInURL(w.Config.Host, loginTestService, "123")
}

// syncBuffer is a bytes.Buffer that the sign-in writes and a fake server reads, from different goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func noopOpen(context.Context, string) error {
	return nil
}

// browserOpens returns a context in which the browser is not disabled, so that SignIn calls the test's Open. The
// runner's BROWSER=none would otherwise skip it.
func browserOpens(ctx context.Context) context.Context {
	return env.Set(ctx, "BROWSER", "test-opener")
}

func (s *loginTestServer) credentialGets() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gets
}

func (s *loginTestServer) isDeleted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleted
}

// loginTestClient starts the fake and returns a client for it.
func loginTestClient(t *testing.T, s *loginTestServer) *databricks.WorkspaceClient {
	t.Helper()
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	return w
}

func TestSignInPollsUntilActive(t *testing.T) {
	s := &loginTestServer{states: []string{"", "PROVISIONING", "ACTIVE"}}
	w := loginTestClient(t, s)

	// The browser is opened in the background, so the URL it is given arrives on a channel.
	opened := make(chan string, 1)
	var out bytes.Buffer
	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open: func(_ context.Context, url string) error {
			opened <- url
			return nil
		},
	})
	require.NoError(t, err)

	wantURL := w.Config.Host + "/mcp-service-login?name=system.ai.github&o=123"
	select {
	case got := <-opened:
		assert.Equal(t, wantURL, got)
	case <-time.After(10 * time.Second):
		t.Fatal("the browser was not opened")
	}
	// The URL is printed before the browser is opened, and polling starts after that.
	assert.Contains(t, out.String(), "Open this URL to sign in to system.ai.github:\n"+wantURL+"\n")
	assert.Equal(t, 3, s.reads)
}

func TestSignInAlreadyActiveDoesNotOpenBrowser(t *testing.T) {
	s := &loginTestServer{states: []string{"ACTIVE"}}
	w := loginTestClient(t, s)

	var out bytes.Buffer
	alreadySignedIn, _, err := SignIn(t.Context(), w, loginTestSignInURL(w), loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: time.Second,
		Open: func(context.Context, string) error {
			t.Fatal("browser must not open when already signed in")
			return nil
		},
	})
	require.NoError(t, err)
	assert.True(t, alreadySignedIn)
	assert.Empty(t, out.String())
}

func TestSignInBrowserNoneOnlyPrintsURL(t *testing.T) {
	t.Setenv("BROWSER", "none")
	s := &loginTestServer{states: []string{"", "ACTIVE"}}
	w := loginTestClient(t, s)

	var out bytes.Buffer
	_, _, err := SignIn(t.Context(), w, loginTestSignInURL(w), loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open: func(context.Context, string) error {
			t.Fatal("browser must not open with BROWSER=none")
			return nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(out.String(), "/mcp-service-login?"), "the URL is printed once")
}

func TestSignInTimeout(t *testing.T) {
	s := &loginTestServer{states: []string{""}}
	w := loginTestClient(t, s)

	var out bytes.Buffer
	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, &out, SignInOptions{
		Open:    noopOpen,
		Poll:    time.Millisecond,
		Timeout: 30 * time.Millisecond,
	})
	timeout := requireSignInTimeout(t, err)
	assert.NoError(t, timeout.LastCheck)
	assert.Contains(t, out.String(), "/mcp-service-login?name=system.ai.github&o=123")
}

func TestSignInCancelledIsNotATimeout(t *testing.T) {
	s := &loginTestServer{states: []string{""}}
	w := loginTestClient(t, s)

	// The caller cancels once the sign-in page is open, so the wait is what is cancelled.
	ctx, cancel := context.WithCancel(t.Context())
	_, _, err := SignIn(browserOpens(ctx), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: time.Minute,
		Open: func(context.Context, string) error {
			cancel()
			return nil
		},
	})
	require.ErrorIs(t, err, context.Canceled)
	_, timedOut := errors.AsType[*SignInTimeoutError](err)
	assert.False(t, timedOut)
}

func TestSignInFailed(t *testing.T) {
	s := &loginTestServer{states: []string{"", "FAILED"}}
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Open:    noopOpen,
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
	})
	require.ErrorIs(t, err, ErrSignInFailed)
}

// requireSignInTimeout checks that err is the timeout of a sign-in, and returns it.
func requireSignInTimeout(t *testing.T, err error) *SignInTimeoutError {
	t.Helper()
	timeout, ok := errors.AsType[*SignInTimeoutError](err)
	require.True(t, ok, "got %v", err)
	return timeout
}

// TestSignInTimeoutReportsLastFailedCheck checks that a timeout after failed polls reports the last of them, so that the
// user can see why the sign-in did not complete.
func TestSignInTimeoutReportsLastFailedCheck(t *testing.T) {
	s := &loginTestServer{states: []string{""}, fail: func(get int) int {
		if get > 1 {
			return http.StatusInternalServerError
		}
		return 0
	}}
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Open:    noopOpen,
		Poll:    time.Millisecond,
		Timeout: 50 * time.Millisecond,
	})
	timeout := requireSignInTimeout(t, err)
	apiErr, ok := errors.AsType[*apierr.APIError](timeout.LastCheck)
	require.True(t, ok, "got %v", timeout.LastCheck)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

// TestSignInTimeoutDoesNotReportAFailedCheckThatWasRecovered checks that a poll which reads the credential after a failed
// one clears the failure, so that a later timeout does not report a check that no longer applies.
func TestSignInTimeoutDoesNotReportAFailedCheckThatWasRecovered(t *testing.T) {
	s := &loginTestServer{states: []string{"", "PROVISIONING"}, fail: func(get int) int {
		if get == 2 {
			return http.StatusInternalServerError
		}
		return 0
	}}
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Open:    noopOpen,
		Poll:    time.Millisecond,
		Timeout: 50 * time.Millisecond,
	})
	assert.NoError(t, requireSignInTimeout(t, err).LastCheck)
}

// TestSignInKeepsPollingThroughServerErrors checks that a poll which fails with a server error does not end the sign-in:
// the user may still be on the provider's page, and the next poll reads the credential.
func TestSignInKeepsPollingThroughServerErrors(t *testing.T) {
	s := &loginTestServer{states: []string{"", "ACTIVE"}, fail: func(get int) int {
		if get == 2 || get == 3 {
			return http.StatusInternalServerError
		}
		return 0
	}}
	w := loginTestClient(t, s)

	alreadySignedIn, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Open:    noopOpen,
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	assert.False(t, alreadySignedIn)
	assert.Equal(t, 4, s.credentialGets(), "the read before the page, two failed polls, and the poll that is ACTIVE")
}

// TestSignInAuthenticationErrorEndsAtOnce checks that a poll which is refused for the caller's credential ends the sign-in
// at once, with that error, rather than polling until the timeout.
func TestSignInAuthenticationErrorEndsAtOnce(t *testing.T) {
	s := &loginTestServer{states: []string{"", "ACTIVE"}, fail: func(get int) int {
		if get == 2 {
			return http.StatusUnauthorized
		}
		return 0
	}}
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Open:    noopOpen,
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
	})
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	require.True(t, ok, "got %v", err)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, 2, s.credentialGets(), "the sign-in ends at the first poll that is refused")
}

func TestCredentialStateNotSignedIn(t *testing.T) {
	s := &loginTestServer{states: []string{""}}
	w := loginTestClient(t, s)

	state, err := CredentialState(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.Empty(t, state)
}

func TestSignOut(t *testing.T) {
	s := &loginTestServer{signedIn: true}
	w := loginTestClient(t, s)

	deleted, err := SignOut(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.True(t, deleted)
	assert.True(t, s.deleted)

	// Once signed out, the server answers NOT_FOUND, which is not an error.
	s.signedIn = false
	deleted, err = SignOut(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.False(t, deleted)
}

// TestSignInDeletesFailedCredentialFromEarlierAttempt checks that a FAILED credential left by an earlier attempt is
// deleted before the page opens, so the FAILED state the wait then sees is not the earlier attempt's, and the new
// sign-in can complete.
func TestSignInDeletesFailedCredentialFromEarlierAttempt(t *testing.T) {
	s := &loginTestServer{states: []string{"FAILED", "", "ACTIVE"}, signedIn: true}
	w := loginTestClient(t, s)

	deletedBeforeOpen := make(chan bool, 1)
	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open: func(context.Context, string) error {
			deletedBeforeOpen <- s.isDeleted()
			return nil
		},
	})
	require.NoError(t, err)
	assert.True(t, <-deletedBeforeOpen)
}

// TestSignInFailedCredentialThatFailsAgainIsFailed checks that a sign-in that fails again, after the FAILED credential
// of an earlier attempt was deleted, is reported as failed at once. Without the deletion the state would never change,
// and the wait would end in a timeout.
func TestSignInFailedCredentialThatFailsAgainIsFailed(t *testing.T) {
	s := &loginTestServer{states: []string{"FAILED"}, signedIn: true}
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 2 * time.Second,
		Open:    noopOpen,
	})
	require.ErrorIs(t, err, ErrSignInFailed)
	assert.True(t, s.isDeleted())
}

// TestSignInFailedAfterProgressIsFailed checks that a FAILED state that follows a change of state fails the sign-in.
func TestSignInFailedAfterProgressIsFailed(t *testing.T) {
	s := &loginTestServer{states: []string{"FAILED", "PROVISIONING", "FAILED"}}
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open:    noopOpen,
	})
	require.ErrorIs(t, err, ErrSignInFailed)
}

// TestSignInTimeoutCutsOffPendingRequest checks that a poll still in flight when the wait ends is cut off, and that the
// end of the wait is ErrSignInTimeout, not the API error of the cut request.
func TestSignInTimeoutCutsOffPendingRequest(t *testing.T) {
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		if reads.Add(1) == 1 {
			// The first read, before the wait, finds no credential.
			http.Error(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`, http.StatusNotFound)
			return
		}
		// A later poll hangs, as a slow workspace does, until the client gives up.
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	start := time.Now()
	_, _, err = SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 200 * time.Millisecond,
		Open:    noopOpen,
	})
	_ = requireSignInTimeout(t, err)
	assert.Less(t, time.Since(start), 5*time.Second)
}

// TestServiceNamesReachTheirOwnCredential checks that a service name with a character that means something in a URL path
// is escaped in the request, so that the request names the credential and never the service. The SDK puts the name into
// the path as it is, so an unescaped "#" would send a DELETE to the service itself.
func TestServiceNamesReachTheirOwnCredential(t *testing.T) {
	for _, name := range []string{"system.ai.git#hub", "system.ai.git?hub", "system.ai.git/hub", "system.ai.git%hub"} {
		t.Run(name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.URL.Path, "/api/") {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				mu.Lock()
				requests = append(requests, r.Method+" "+r.URL.EscapedPath())
				mu.Unlock()
				http.Error(rw, `{"error_code":"NOT_FOUND","message":"not found"}`, http.StatusNotFound)
			}))
			t.Cleanup(server.Close)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)

			_, err = SignOut(t.Context(), w, name)
			require.NoError(t, err)
			_, err = CredentialState(t.Context(), w, name)
			require.NoError(t, err)
			_, err = GetService(t.Context(), w, name)
			require.ErrorIs(t, err, apierr.ErrNotFound)

			escaped := "/api/2.1/unity-catalog/mcp-services/" + url.PathEscape(name)
			assert.Equal(t, []string{
				"DELETE " + escaped + "/user-credentials",
				"GET " + escaped + "/user-credentials",
				"GET " + escaped,
			}, requests)
		})
	}
}

// TestEndpointNotFoundIsNotTheAnswerAboutTheCredential checks that a 404 for an endpoint the API does not have is an
// error. It does not mean that the caller is not signed in, or that the credential is gone.
func TestEndpointNotFoundIsNotTheAnswerAboutTheCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		http.Error(rw, `{"error_code":"ENDPOINT_NOT_FOUND","message":"endpoint is not available"}`, http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	state, err := CredentialState(t.Context(), w, loginTestService)
	require.Error(t, err)
	assert.Empty(t, state)

	deleted, err := SignOut(t.Context(), w, loginTestService)
	require.Error(t, err)
	assert.False(t, deleted)
}

// TestSignInDoesNotWaitForTheOpener checks that a browser command still running does not hold up the wait for the
// sign-in. The command can stay open as long as the browser is.
func TestSignInDoesNotWaitForTheOpener(t *testing.T) {
	s := &loginTestServer{states: []string{"", "ACTIVE"}}
	w := loginTestClient(t, s)

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	done := make(chan error, 1)
	go func() {
		_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
			Poll:    time.Millisecond,
			Timeout: 10 * time.Second,
			Open: func(context.Context, string) error {
				<-release
				return nil
			},
		})
		done <- err
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the sign-in waited for the browser command to return")
	}
}

// TestSignInReportsOpenerFailure checks that an opener that fails is reported while the wait goes on. The URL is printed
// before the page opens, so the user can still finish the sign-in by hand.
func TestSignInReportsOpenerFailure(t *testing.T) {
	var out syncBuffer
	s := &loginTestServer{states: []string{"ACTIVE"}}
	// The credential becomes ACTIVE only after the failure is printed, so the wait cannot end before the message.
	s.gate = func() bool { return strings.Contains(out.String(), "Could not open a browser") }
	w := loginTestClient(t, s)

	_, _, err := SignIn(browserOpens(t.Context()), w, loginTestSignInURL(w), loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open: func(context.Context, string) error {
			return errors.New("no browser here")
		},
	})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Could not open a browser (no browser here). Open the URL above.\n")
}

// TestSignInCallsOnPollOnEachPoll checks that OnPoll is called once for each poll of the wait, after the first read of the
// credential, which is not a poll.
func TestSignInCallsOnPollOnEachPoll(t *testing.T) {
	s := &loginTestServer{states: []string{"", "PROVISIONING", "ACTIVE"}}
	w := loginTestClient(t, s)
	polls := 0
	_, _, err := SignIn(t.Context(), w, loginTestSignInURL(w), loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open:    func(context.Context, string) error { return nil },
		OnPoll:  func() { polls++ },
	})
	require.NoError(t, err)
	assert.Equal(t, 2, polls)
	assert.Equal(t, 3, s.reads)
}
