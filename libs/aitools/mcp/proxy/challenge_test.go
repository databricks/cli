package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testService = "system.ai.github"
	testHost    = "https://workspace.test"
	testURL     = testHost + "/mcp-service-login?name=system.ai.github&o=1"
	// Each message is one line, so an upstream body that passes through is byte-equal on stdout. An upstream answer
	// carries the id of its request, so each request id has its own fixtures.
	listChallenge  = `{"jsonrpc":"2.0","id":2,"error":{"code":-32006,"message":"rejected","data":null}}`
	callChallenge  = `{"jsonrpc":"2.0","id":3,"error":{"code":-32006,"message":"rejected","data":null}}`
	listTools      = `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","inputSchema":{"type":"object"}}]}}`
	listToolsLater = `{"jsonrpc":"2.0","id":4,"result":{"tools":[{"name":"echo","inputSchema":{"type":"object"}}]}}`
	listChangedMsg = `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`
)

// signInStub replaces the command's sign-in. Each call records its request and returns reply(call), with the page
// reported as shown.
type signInStub struct {
	mu    sync.Mutex
	reqs  []SignInRequest
	reply func(call int) (bool, error)
}

func (s *signInStub) SignIn(_ context.Context, req SignInRequest) (bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs = append(s.reqs, req)
	alreadySignedIn, err := s.reply(len(s.reqs))
	return alreadySignedIn, true, err
}

// calls returns whether each sign-in opened the page.
func (s *signInStub) calls() []bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	var opens []bool
	for _, req := range s.reqs {
		opens = append(opens, req.Open)
	}
	return opens
}

// requests returns the request of each sign-in.
func (s *signInStub) requests() []SignInRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]SignInRequest(nil), s.reqs...)
}

// loginCommand is the hint that the command builds for a pinned config file and a profile name with a space.
const loginCommand = `DATABRICKS_CONFIG_FILE='/work/cfg/my file.cfg' databricks aitools mcp login system.ai.github --profile 'my profile'`

// challengedBridge returns a bridge to url with the sign-in hooks of a service that needs sign-in.
func challengedBridge(url string, signIn SignInFunc) *Bridge {
	b := bridgeFor(url)
	b.Hooks = newSignInHooks(&Challenge{Service: testService, Profile: "logfood", Host: testHost, Label: "GitHub", LoginCommand: "databricks aitools mcp login system.ai.github --profile logfood", SignIn: signIn}, io.Discard)
	return b
}

// answerWith replies to every request with status and body.
func answerWith(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

// challengedService answers tools/list with the sign-in challenge and every other request with 202, as a service that
// needs sign-in does until the caller signs in.
func challengedService(t *testing.T) (*httptest.Server, *fakeEndpoint) {
	return newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if m, _ := parseMessage(body); m.Method != "tools/list" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		answerWith(http.StatusOK, listChallenge)(w, r)
	})
}

// challenge lists the tools of the session, which the challengedService answers with the challenge, and waits for the
// answer. The calls after it see a challenged session.
func challenge(t *testing.T, p *piped) {
	t.Helper()
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	p.waitLines(t, 1)
}

// unauthorized answers 401 with a plain body, as the gateway does to a caller it cannot authenticate.
func unauthorized(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

// resultText returns the text of a tool call result line.
func resultText(t *testing.T, line string) string {
	t.Helper()
	return parseReply(t, line).Result["content"].([]any)[0].(map[string]any)["text"].(string)
}

// toolNames returns the tool names of a tools/list result line.
func toolNames(t *testing.T, line string) []string {
	t.Helper()
	reply := parseReply(t, line)
	var names []string
	for _, tool := range reply.Result["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	return names
}

func TestChallengeReplacesListWithAuthenticateTool(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"rpc -32006": answerWith(http.StatusOK, listChallenge),
		"http 401":   unauthorized,
		"url elicitation": answerWith(http.StatusOK, `{"jsonrpc":"2.0","id":2,"error":{"code":-32042,"message":"x","data":{"elicitations":[`+
			`{"mode":"url","url":"`+testURL+`"}]}}}`),
	}
	for name, handle := range cases {
		t.Run(name, func(t *testing.T) {
			srv, _ := newFakeEndpoint(t, handle)
			signIn := &signInStub{reply: func(int) (bool, error) { return false, nil }}
			out, _ := runAll(t, challengedBridge(srv.URL, signIn.SignIn), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

			require.Len(t, out, 1)
			assert.Equal(t, []string{"authenticate"}, toolNames(t, out[0]))
			reply := parseReply(t, out[0])
			assert.JSONEq(t, `2`, string(reply.ID))
			assert.Empty(t, signIn.calls(), "tools/list must not start a sign-in")
		})
	}
}

func TestChallengeListPassesThroughWhenSignedIn(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, listTools))
	out, _ := runAll(t, challengedBridge(srv.URL, nil), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{listTools}, out)
}

func TestChallengeIgnoresOtherErrors(t *testing.T) {
	cases := map[string]string{
		"generic error": `{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"boom"}}`,
		"mixed elicitation": `{"jsonrpc":"2.0","id":2,"error":{"code":-32042,"message":"x","data":{"elicitations":[` +
			`{"mode":"url","url":"` + testURL + `"},{"mode":"form"}]}}}`,
		"other host": `{"jsonrpc":"2.0","id":2,"error":{"code":-32042,"message":"x","data":{"elicitations":[` +
			`{"mode":"url","url":"https://elsewhere.test/mcp-service-login?name=system.ai.github"}]}}}`,
		"other service": `{"jsonrpc":"2.0","id":2,"error":{"code":-32042,"message":"x","data":{"elicitations":[` +
			`{"mode":"url","url":"` + testHost + `/mcp-service-login?name=system.ai.slack"}]}}}`,
		"other path": `{"jsonrpc":"2.0","id":2,"error":{"code":-32042,"message":"x","data":{"elicitations":[` +
			`{"mode":"url","url":"` + testHost + `/oidc?name=system.ai.github"}]}}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, body))
			signIn := &signInStub{reply: func(int) (bool, error) { return false, nil }}
			out, _ := runAll(t, challengedBridge(srv.URL, signIn.SignIn), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

			assert.Equal(t, []string{body}, out)
			assert.Empty(t, signIn.calls())
		})
	}
}

func TestChallengeToolsCallAnswersWithoutBrowser(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, callChallenge))
	signIn := &signInStub{reply: func(int) (bool, error) { return false, nil }}
	out, _ := runAll(t, challengedBridge(srv.URL, signIn.SignIn),
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`+"\n")

	require.Len(t, out, 2)
	reply := parseReply(t, out[0])
	assert.JSONEq(t, `3`, string(reply.ID))
	text := resultText(t, out[0])
	assert.Equal(t, true, reply.Result["isError"])
	assert.Contains(t, text, "system.ai.github on profile logfood needs a one-time sign-in to GitHub.")
	assert.Contains(t, text, "databricks aitools mcp login system.ai.github --profile logfood")
	assert.Equal(t, listChangedMsg, out[1])
	assert.Empty(t, signIn.calls(), "a challenged tools/call must not open the sign-in page")
}

func TestChallengeToolsCallWithSignInURLInTextIsChallenged(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":3,"result":{"isError":true,"content":[{"type":"text","text":"Sign in first: ` + testURL + `."}]}}`
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, body))
	out, _ := runAll(t, challengedBridge(srv.URL, nil),
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`+"\n")

	require.Len(t, out, 2)
	assert.Equal(t, listChangedMsg, out[1])
	assert.Contains(t, out[0], "needs a one-time sign-in")
}

func TestChallengeNeverTriggeredByInitialize(t *testing.T) {
	srv, _ := newFakeEndpoint(t, unauthorized)
	out, _ := runAll(t, challengedBridge(srv.URL, nil), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n")

	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	require.NotNil(t, reply.Error)
	assert.Equal(t, codeUpstream, reply.Error.Code)
	assert.Nil(t, reply.Result)
}

func TestAuthenticateSignsInAndSendsListChanged(t *testing.T) {
	srv, fake := challengedService(t)
	signIn := &signInStub{reply: func(int) (bool, error) { return false, nil }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	challenge(t, p)
	p.send(t, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	lines := p.waitLines(t, 3)
	require.NoError(t, p.finish(t))

	assert.Equal(t, []bool{true}, signIn.calls())
	assert.Len(t, fake.requests(), 1, "authenticate is answered locally, never sent upstream; only the list is")
	reply := parseReply(t, lines[1])
	assert.JSONEq(t, `4`, string(reply.ID))
	assert.Equal(t, false, reply.Result["isError"])
	assert.Contains(t, lines[1], "Signed in to system.ai.github on profile logfood. Tools are available now")
	assert.Equal(t, listChangedMsg, lines[2])
}

func TestAuthenticateSaysAlreadySignedInAfterSignIn(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := &signInStub{reply: func(call int) (bool, error) { return call > 1, nil }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
	challenge(t, p)

	p.send(t, fmt.Sprintf(authenticate, 5))
	p.waitLines(t, 3)
	p.send(t, fmt.Sprintf(authenticate, 6))
	lines := p.waitLines(t, 5)

	assert.Equal(t, []bool{true, true}, signIn.calls(), "a sign-in that completed does not hold the page back")
	assert.Contains(t, lines[3], "Already signed in to system.ai.github")
	assert.Equal(t, listChangedMsg, lines[4], "an agent whose list still shows only authenticate is asked to list again")
	require.NoError(t, p.finish(t))
}

func TestAuthenticateReopensPageAfterTimeout(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := &signInStub{reply: func(call int) (bool, error) {
		if call == 1 {
			return false, &mcp.SignInTimeoutError{URL: testURL, Wait: 5 * time.Minute}
		}
		return false, nil
	}}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
	challenge(t, p)

	p.send(t, fmt.Sprintf(authenticate, 7))
	lines := p.waitLines(t, 2)
	require.Len(t, lines, 2)
	assert.Equal(t, true, parseReply(t, lines[1]).Result["isError"])
	text := resultText(t, lines[1])
	assert.Contains(t, text, "did not complete in 5 minutes. Finish at "+testURL)
	assert.Contains(t, text, "databricks aitools mcp login system.ai.github --profile logfood")

	p.send(t, fmt.Sprintf(authenticate, 8))
	lines = p.waitLines(t, 4)

	assert.Equal(t, []bool{true, true}, signIn.calls(), "a timed-out attempt may open the page again")
	assert.Contains(t, lines[2], "Signed in to system.ai.github")
	require.NoError(t, p.finish(t))
}

func TestAuthenticateReportsFailureWithoutListChanged(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := &signInStub{reply: func(int) (bool, error) { return false, errors.New("sign-in failed for the test") }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	challenge(t, p)
	p.send(t, `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	lines := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	require.Len(t, lines, 2, "the failure is the only answer to authenticate, and no list_changed follows it")
	assert.Equal(t, true, parseReply(t, lines[1]).Result["isError"])
	assert.Contains(t, lines[1], "Sign-in to system.ai.github on profile logfood failed: sign-in failed for the test")
}

// TestChallengeTurnsOnAfterSignIn runs the flow an agent sees: a challenged list, authenticate, then the
// list passes through once the credential exists upstream.
func TestChallengeTurnsOnAfterSignIn(t *testing.T) {
	var signedIn atomic.Bool
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if signedIn.Load() {
			answerWith(http.StatusOK, listToolsLater)(w, r)
			return
		}
		answerWith(http.StatusOK, listChallenge)(w, r)
	})
	signIn := &signInStub{reply: func(int) (bool, error) {
		signedIn.Store(true)
		return false, nil
	}}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))

	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	lines := p.waitLines(t, 1)
	assert.Equal(t, []string{"authenticate"}, toolNames(t, lines[0]))

	p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	lines = p.waitLines(t, 3)
	assert.Contains(t, lines[1], "Signed in to system.ai.github")
	assert.Equal(t, listChangedMsg, lines[2])

	p.send(t, `{"jsonrpc":"2.0","id":4,"method":"tools/list"}`)
	lines = p.waitLines(t, 4)
	assert.Equal(t, []string{"echo"}, toolNames(t, lines[3]))
	require.NoError(t, p.finish(t))
}

// TestChallengeHintsPrintTheLoginCommand checks that the challenge and the timeout print the login command that the
// command built, so a pinned config file and a profile name with a space are quoted for a shell.
func TestChallengeHintsPrintTheLoginCommand(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, callChallenge))
	var calls atomic.Int32
	signIn := func(ctx context.Context, _ SignInRequest) (bool, bool, error) {
		calls.Add(1)
		return false, true, &mcp.SignInTimeoutError{URL: testURL, Wait: 5 * time.Minute}
	}
	b := bridgeFor(srv.URL)
	b.Hooks = newSignInHooks(&Challenge{Service: testService, Profile: "my profile", Host: testHost, Label: "GitHub", LoginCommand: loginCommand, SignIn: signIn}, io.Discard)
	p := startPiped(t, t.Context(), b)

	p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`)
	lines := p.waitLines(t, 2)
	assert.Contains(t, resultText(t, lines[0]), "Call the authenticate tool, or run: "+loginCommand+". If")

	p.send(t, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	lines = p.waitLines(t, 3)
	require.NoError(t, p.finish(t))
	assert.EqualValues(t, 1, calls.Load())
	assert.Contains(t, resultText(t, lines[2]), "then retry, or run: "+loginCommand)
}

// TestAuthenticateWritesNothingWhenCancelled checks that an authenticate call the client cancelled is not answered: the
// MCP spec has no response to a cancelled request, and the client has stopped waiting for one.
func TestAuthenticateWritesNothingWhenCancelled(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if m, _ := parseMessage(body); m.Method != "tools/list" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		answerWith(http.StatusOK, listChallenge)(w, r)
	})
	entered, returned := make(chan struct{}), make(chan struct{})
	signIn := func(ctx context.Context, _ SignInRequest) (bool, bool, error) {
		close(entered)
		<-ctx.Done()
		close(returned)
		return false, true, ctx.Err()
	}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))

	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	waitClosed(t, entered, "the sign-in to start")
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":5}}`)
	waitClosed(t, returned, "the sign-in to return")

	require.NoError(t, p.finish(t))
	lines := outLines(p.out.String())
	require.Len(t, lines, 1, "only the tools/list answer is written")
	assert.Equal(t, []string{"authenticate"}, toolNames(t, lines[0]))
}

// attempt is the outcome of one sign-in in a test. A cancelled attempt waits for its context, as a call the client
// cancelled does, and then returns the context's error.
type attempt struct {
	cancel          bool
	urlShown        bool
	alreadySignedIn bool
	err             error
}

// TestAuthenticateOpensPageAgainUnlessACancelledAttemptShowedIt checks when a later authenticate call opens the sign-in
// page again. The page is held back only while a cancelled attempt that showed it may still be open in the user's
// browser. A timeout, a failure, an answer that needs no sign-in, a success that is later revoked, and a cancelled
// attempt that never showed the page all open it again.
func TestAuthenticateOpensPageAgainUnlessACancelledAttemptShowedIt(t *testing.T) {
	cases := map[string]struct {
		first    attempt
		wantOpen bool
	}{
		"cancelled after the page was shown":  {first: attempt{cancel: true, urlShown: true}, wantOpen: false},
		"cancelled before the page was shown": {first: attempt{cancel: true}, wantOpen: true},
		"timed out":                           {first: attempt{urlShown: true, err: &mcp.SignInTimeoutError{URL: testURL, Wait: time.Minute}}, wantOpen: true},
		"failed":                              {first: attempt{urlShown: true, err: errors.New("sign-in failed")}, wantOpen: true},
		"error before the page":               {first: attempt{err: errors.New("credential read failed")}, wantOpen: true},
		"already signed in":                   {first: attempt{alreadySignedIn: true, urlShown: true}, wantOpen: true},
		"signed in, then revoked":             {first: attempt{urlShown: true}, wantOpen: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, listChallenge))
			entered, done := make(chan struct{}), make(chan struct{})
			var mu sync.Mutex
			var opens []bool
			signIn := func(ctx context.Context, req SignInRequest) (bool, bool, error) {
				mu.Lock()
				opens = append(opens, req.Open)
				first := len(opens) == 1
				mu.Unlock()
				if !first {
					return false, true, nil
				}
				if !tc.first.cancel {
					return tc.first.alreadySignedIn, tc.first.urlShown, tc.first.err
				}
				close(entered)
				<-ctx.Done()
				close(done)
				return false, tc.first.urlShown, ctx.Err()
			}
			p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
			authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`

			p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
			p.waitLines(t, 1)
			p.send(t, fmt.Sprintf(authenticate, 5))
			if tc.first.cancel {
				waitClosed(t, entered, "the first sign-in to start")
				p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":5}}`)
				waitClosed(t, done, "the first sign-in to return")
			}
			p.send(t, fmt.Sprintf(authenticate, 6))
			require.Eventually(t, func() bool {
				mu.Lock()
				defer mu.Unlock()
				return len(opens) == 2
			}, 10*time.Second, time.Millisecond)
			require.NoError(t, p.finish(t))

			mu.Lock()
			defer mu.Unlock()
			require.Len(t, opens, 2)
			assert.True(t, opens[0], "the first attempt opens the page")
			assert.Equal(t, tc.wantOpen, opens[1])
		})
	}
}

// TestAuthenticateIsForwardedBeforeAChallenge checks that before the service challenges the session, a tool named
// authenticate is the upstream's tool and is forwarded. The proxy's tool exists only after a challenge.
func TestAuthenticateIsForwardedBeforeAChallenge(t *testing.T) {
	const upstream = `{"jsonrpc":"2.0","id":4,"result":{"content":[{"type":"text","text":"upstream"}]}}`
	srv, fake := newFakeEndpoint(t, answerWith(http.StatusOK, upstream))
	signIn := &signInStub{reply: func(int) (bool, error) { return false, nil }}
	out, _ := runAll(t, challengedBridge(srv.URL, signIn.SignIn),
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`+"\n")

	assert.Empty(t, signIn.calls(), "no sign-in runs for a tool the service answers")
	assert.Len(t, fake.requests(), 1)
	assert.Equal(t, []string{upstream}, out)
}

// TestChallengeMatchesServiceNameWithoutCase checks that a service named in another case is challenged by its sign-in
// URL. Binding and Unity Catalog names are compared without case, so `--service SYSTEM.AI.GITHUB` is a service too.
func TestChallengeMatchesServiceNameWithoutCase(t *testing.T) {
	const body = `{"jsonrpc":"2.0","id":2,"error":{"code":-32042,"message":"x","data":{"elicitations":[` +
		`{"mode":"url","url":"` + testURL + `"}]}}}`
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, body))
	b := bridgeFor(srv.URL)
	b.Hooks = newSignInHooks(&Challenge{Service: "SYSTEM.AI.GITHUB", Profile: "logfood", Host: testHost, Label: "GitHub", LoginCommand: loginCommand}, io.Discard)
	out, _ := runAll(t, b, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	assert.Equal(t, []string{"authenticate"}, toolNames(t, out[0]))
}

// TestChallengeTextAroundSignInURL checks that a sign-in URL in the text of an error result is recognized whatever
// punctuation surrounds it, as a sentence or a list puts it around the URL.
func TestChallengeTextAroundSignInURL(t *testing.T) {
	for _, around := range []struct{ before, after string }{
		{"Sign in first: ", "."},
		{"Sign in (", ")"},
		{"Sign in [", "]"},
		{`Sign in "`, `".`},
		{"Sign in '", "';"},
		{"[Sign in](", ")"},
		{"<", ">"},
		{"**", "**"},
		{"Open **", "** now."},
	} {
		t.Run(around.before+"URL"+around.after, func(t *testing.T) {
			text, err := json.Marshal(around.before + testURL + around.after)
			require.NoError(t, err)
			body := `{"jsonrpc":"2.0","id":3,"result":{"isError":true,"content":[{"type":"text","text":` + string(text) + `}]}}`
			srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, body))
			out, _ := runAll(t, challengedBridge(srv.URL, nil),
				`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`+"\n")

			require.Len(t, out, 2)
			assert.Equal(t, listChangedMsg, out[1])
			assert.Contains(t, out[0], "needs a one-time sign-in")
		})
	}
}

// TestAuthenticateStartsSignInAfterChallengeFollowingAlreadySignedIn checks that when the service challenges again after
// an "Already signed in" answer, the next authenticate starts a new sign-in from no credential. The credential reads
// ACTIVE, but the service does not accept it, so answering "Already signed in" again would loop.
func TestAuthenticateStartsSignInAfterChallengeFollowingAlreadySignedIn(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := &signInStub{reply: func(call int) (bool, error) { return call == 1, nil }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
	challenge(t, p)

	// "Already signed in" is followed by tools/list_changed.
	p.send(t, fmt.Sprintf(authenticate, 5))
	p.waitLines(t, 3)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	p.waitLines(t, 4)
	p.send(t, fmt.Sprintf(authenticate, 7))
	lines := p.waitLines(t, 5)
	require.NoError(t, p.finish(t))

	reqs := signIn.requests()
	require.Len(t, reqs, 2)
	assert.False(t, reqs[0].Restart)
	assert.True(t, reqs[1].Restart, "the sign-in after the challenge starts from no credential")
	assert.Contains(t, lines[1], "Already signed in to system.ai.github")
	assert.Equal(t, listChangedMsg, lines[2])
	assert.Contains(t, lines[4], "Signed in to system.ai.github")
}

// TestAuthenticateStaleAnswerWithoutChallengeKeepsCredential checks that a second authenticate that is not preceded by a
// challenge does not delete the credential, so it answers "Already signed in" again.
func TestAuthenticateStaleAnswerWithoutChallengeKeepsCredential(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := &signInStub{reply: func(int) (bool, error) { return true, nil }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
	challenge(t, p)

	p.send(t, fmt.Sprintf(authenticate, 5))
	p.waitLines(t, 2)
	p.send(t, fmt.Sprintf(authenticate, 6))
	p.waitLines(t, 3)
	require.NoError(t, p.finish(t))

	for _, req := range signIn.requests() {
		assert.False(t, req.Restart)
	}
}

// TestAuthenticateSendsProgressOnEachPoll checks that an authenticate call with a progress token gets a progress
// notification on each poll of its wait, numbered in order, before its answer. Progress keeps a client that times out
// requests without progress from giving up while the user signs in.
func TestAuthenticateSendsProgressOnEachPoll(t *testing.T) {
	srv, _ := challengedService(t)
	proceed := make(chan struct{})
	signIn := func(_ context.Context, req SignInRequest) (bool, bool, error) {
		for range 2 {
			req.Progress()
			<-proceed
		}
		return false, true, nil
	}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
	challenge(t, p)

	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{},"_meta":{"progressToken":"sign-in-1"}}}`)
	lines := p.waitLines(t, 2)
	assert.JSONEq(t, `{"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"sign-in-1","progress":1,"message":"Waiting for the sign-in to GitHub to complete."}}`, lines[1])
	proceed <- struct{}{}

	lines = p.waitLines(t, 3)
	assert.JSONEq(t, `{"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"sign-in-1","progress":2,"message":"Waiting for the sign-in to GitHub to complete."}}`, lines[2])
	proceed <- struct{}{}

	lines = p.waitLines(t, 5)
	require.NoError(t, p.finish(t))
	assert.Contains(t, lines[3], "Signed in to system.ai.github")
	assert.Equal(t, listChangedMsg, lines[4])
}

// TestAuthenticateSendsNoProgressWithoutToken checks that a call without a progress token gets no progress notification.
func TestAuthenticateSendsNoProgressWithoutToken(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := func(_ context.Context, req SignInRequest) (bool, bool, error) {
		req.Progress()
		req.Progress()
		return false, true, nil
	}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
	challenge(t, p)
	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	lines := p.waitLines(t, 3)
	require.NoError(t, p.finish(t))

	require.Len(t, lines, 3)
	assert.NotContains(t, strings.Join(lines, "\n"), "notifications/progress")
}

// TestAuthenticateSendsNoProgressForNullToken checks that a progressToken of null is no token: the client asked for no
// progress, so none is sent.
func TestAuthenticateSendsNoProgressForNullToken(t *testing.T) {
	srv, _ := challengedService(t)
	signIn := func(_ context.Context, req SignInRequest) (bool, bool, error) {
		req.Progress()
		return false, true, nil
	}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
	challenge(t, p)
	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{},"_meta":{"progressToken":null}}}`)
	lines := p.waitLines(t, 3)
	require.NoError(t, p.finish(t))

	assert.NotContains(t, strings.Join(lines, "\n"), "notifications/progress")
}

// listAnswer answers each tools/list with the challenge while challenged is set, and with one tool otherwise. A tools/call
// is answered by the upstream's tool, as a signed-in service does. Unlike the fixtures above it answers with the id of
// each request, so later requests match their answers.
func listAnswer(challenged *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case m.Method == "tools/list" && challenged.Load():
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32006,"message":"rejected","data":null}}`, m.ID)
		case m.Method == "tools/list":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[{"name":"echo","inputSchema":{"type":"object"}}]}}`, m.ID)
		case m.Method == methodToolsCall:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":"upstream"}],"isError":false}}`, m.ID)
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

// TestChallengeAfterSuccessDoesNotRestartSignIn checks that a challenge long after an "Already signed in" answer does not
// start the next sign-in from no credential. A successful answer from the service ends that state, so only a challenge
// that follows the answer with no success in between does.
func TestChallengeAfterSuccessDoesNotRestartSignIn(t *testing.T) {
	var challenged atomic.Bool
	challenged.Store(true)
	srv, _ := newFakeEndpoint(t, listAnswer(&challenged))
	signIn := &signInStub{reply: func(int) (bool, error) { return true, nil }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
	challenge(t, p)

	p.send(t, fmt.Sprintf(authenticate, 5))
	p.waitLines(t, 3)
	challenged.Store(false)
	p.send(t, `{"jsonrpc":"2.0","id":6,"method":"tools/list"}`)
	p.waitLines(t, 4)
	challenged.Store(true)
	p.send(t, `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`)
	p.waitLines(t, 5)
	p.send(t, fmt.Sprintf(authenticate, 8))
	p.waitLines(t, 6)
	require.NoError(t, p.finish(t))

	reqs := signIn.requests()
	require.Len(t, reqs, 2)
	assert.False(t, reqs[1].Restart, "a challenge after a success is not a challenge to the credential that was reported signed in")
}

// TestRestartIsNotRepeatedAfterItsPageWasShown checks the restart that a challenge after "Already signed in" asks for.
// Once its page was shown the credential was deleted, and a sign-in cancelled after that does not delete again, since
// the user may finish the page that is still open. A restart whose page was never shown is asked for again.
func TestRestartIsNotRepeatedAfterItsPageWasShown(t *testing.T) {
	for name, urlShown := range map[string]bool{"page shown": true, "page not shown": false} {
		t.Run(name, func(t *testing.T) {
			var challenged atomic.Bool
			challenged.Store(true)
			srv, _ := newFakeEndpoint(t, listAnswer(&challenged))
			entered, returned := make(chan struct{}), make(chan struct{})
			var mu sync.Mutex
			var reqs []SignInRequest
			signIn := func(ctx context.Context, req SignInRequest) (bool, bool, error) {
				mu.Lock()
				reqs = append(reqs, req)
				n := len(reqs)
				mu.Unlock()
				switch n {
				case 1:
					return true, false, nil
				case 2:
					close(entered)
					<-ctx.Done()
					close(returned)
					return false, urlShown, ctx.Err()
				default:
					return false, true, nil
				}
			}
			p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
			authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
			challenge(t, p)

			p.send(t, fmt.Sprintf(authenticate, 5))
			p.waitLines(t, 3)
			p.send(t, `{"jsonrpc":"2.0","id":6,"method":"tools/list"}`)
			p.waitLines(t, 4)
			p.send(t, fmt.Sprintf(authenticate, 7))
			waitClosed(t, entered, "the restart to start")
			p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7}}`)
			waitClosed(t, returned, "the restart to return")
			p.send(t, fmt.Sprintf(authenticate, 8))
			p.waitLines(t, 5)
			require.NoError(t, p.finish(t))

			mu.Lock()
			defer mu.Unlock()
			require.Len(t, reqs, 3)
			assert.True(t, reqs[1].Restart, "the challenge after the answer asks for a restart")
			assert.Equal(t, !urlShown, reqs[2].Restart)
		})
	}
}

// TestAuthenticateIsForwardedOnceTheListPasses checks that the authenticate tool is the proxy's only while its list is
// shown. Once a tools/list passes through unchallenged, the list is the service's again, and a tool of the service named
// authenticate is forwarded to it.
func TestAuthenticateIsForwardedOnceTheListPasses(t *testing.T) {
	var challenged atomic.Bool
	challenged.Store(true)
	srv, _ := newFakeEndpoint(t, listAnswer(&challenged))
	signIn := &signInStub{reply: func(int) (bool, error) {
		challenged.Store(false)
		return false, nil
	}}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`
	challenge(t, p)

	p.send(t, fmt.Sprintf(authenticate, 5))
	p.waitLines(t, 3)
	p.send(t, `{"jsonrpc":"2.0","id":6,"method":"tools/list"}`)
	p.waitLines(t, 4)
	p.send(t, fmt.Sprintf(authenticate, 7))
	lines := p.waitLines(t, 5)
	require.NoError(t, p.finish(t))

	assert.Equal(t, []bool{true}, signIn.calls(), "the forwarded call starts no sign-in")
	assert.Equal(t, `{"jsonrpc":"2.0","id":7,"result":{"content":[{"type":"text","text":"upstream"}],"isError":false}}`, lines[4])
}

// TestQueuedAuthenticateCancelledTouchesNothing checks that an authenticate call the client cancels while it waits for
// the sign-in ahead of it neither answers nor starts a sign-in, and does not clear the record that the page was cancelled.
// The next call therefore does not open the page again.
func TestQueuedAuthenticateCancelledTouchesNothing(t *testing.T) {
	srv, _ := challengedService(t)
	entered, returned := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	var opens []bool
	signIn := func(ctx context.Context, req SignInRequest) (bool, bool, error) {
		mu.Lock()
		opens = append(opens, req.Open)
		first := len(opens) == 1
		mu.Unlock()
		if !first {
			return false, false, nil
		}
		close(entered)
		<-ctx.Done()
		close(returned)
		return false, true, ctx.Err()
	}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
	challenge(t, p)

	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	waitClosed(t, entered, "the first sign-in to start")
	p.send(t, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":6}}`)
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":5}}`)
	waitClosed(t, returned, "the first sign-in to return")
	p.send(t, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(opens) == 2
	}, 10*time.Second, time.Millisecond)
	require.NoError(t, p.finish(t))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []bool{true, false}, opens, "the call after the cancelled ones keeps the page closed")
	out := p.out.String()
	assert.NotContains(t, out, `"id":5`, "a cancelled call is not answered")
	assert.NotContains(t, out, `"id":6`, "a call cancelled while it waits is not answered")
}

// TestQueuedAuthenticateReportsProgress checks that an authenticate call waiting for the sign-in ahead of it reports
// progress on its progress token, so that a client that times out requests without progress does not give up on it.
func TestQueuedAuthenticateReportsProgress(t *testing.T) {
	prev := queueProgressInterval
	queueProgressInterval = time.Millisecond
	t.Cleanup(func() { queueProgressInterval = prev })
	srv, _ := challengedService(t)
	entered, release := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	calls := 0
	signIn := func(context.Context, SignInRequest) (bool, bool, error) {
		mu.Lock()
		calls++
		first := calls == 1
		mu.Unlock()
		if first {
			close(entered)
			<-release
			return false, true, nil
		}
		return true, false, nil
	}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn))
	challenge(t, p)

	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	waitClosed(t, entered, "the first sign-in to start")
	p.send(t, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"authenticate","arguments":{},"_meta":{"progressToken":"queued"}}}`)
	require.Eventually(t, func() bool {
		return strings.Contains(p.out.String(), `"progressToken":"queued"`)
	}, 10*time.Second, time.Millisecond)
	close(release)
	require.Eventually(t, func() bool {
		return strings.Contains(p.out.String(), "Already signed in")
	}, 10*time.Second, time.Millisecond)
	require.NoError(t, p.finish(t))

	lines := outLines(p.out.String())
	var queued, signedIn int
	for _, line := range lines {
		if strings.Contains(line, `"progressToken":"queued"`) {
			queued++
		}
		if strings.Contains(line, "Signed in to") {
			signedIn = queued
		}
	}
	assert.Positive(t, signedIn, "the queued call's progress comes before the first sign-in answers")
}
