package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testService = "system.ai.github"
	testHost    = "https://workspace.test"
	testURL     = testHost + "/mcp-service-login?name=system.ai.github&o=1"
	// Each message is one line, so an upstream body that passes through is byte-equal on stdout.
	listChallenge  = `{"jsonrpc":"2.0","id":2,"error":{"code":-32006,"message":"rejected","data":null}}`
	listTools      = `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","inputSchema":{"type":"object"}}]}}`
	listChangedMsg = `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`
)

// signInStub replaces the command's sign-in. Each call records its open flag and returns reply(call).
type signInStub struct {
	mu    sync.Mutex
	opens []bool
	reply func(call int) (bool, error)
}

func (s *signInStub) SignIn(_ context.Context, open bool) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opens = append(s.opens, open)
	return s.reply(len(s.opens))
}

func (s *signInStub) calls() []bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]bool(nil), s.opens...)
}

// challengedBridge returns a bridge to url with the sign-in hooks of a service that needs sign-in.
func challengedBridge(url string, signIn SignInFunc) *Bridge {
	b := bridgeFor(url)
	b.Hooks = &signInHooks{c: &Challenge{Service: testService, Profile: "logfood", Host: testHost, Label: "GitHub", SignIn: signIn}}
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
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, listChallenge))
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
	srv, fake := newFakeEndpoint(t, answerWith(http.StatusOK, listTools))
	signIn := &signInStub{reply: func(int) (bool, error) { return false, nil }}
	out, _ := runAll(t, challengedBridge(srv.URL, signIn.SignIn),
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`+"\n")

	require.Len(t, out, 2)
	assert.Equal(t, []bool{true}, signIn.calls())
	assert.Empty(t, fake.requests(), "authenticate is answered locally, never sent upstream")
	reply := parseReply(t, out[0])
	assert.JSONEq(t, `4`, string(reply.ID))
	assert.Equal(t, false, reply.Result["isError"])
	assert.Contains(t, out[0], "Signed in to system.ai.github on profile logfood. Tools are available now")
	assert.Equal(t, listChangedMsg, out[1])
}

func TestAuthenticateOpensPageOnceAndSaysAlreadySignedIn(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, listTools))
	signIn := &signInStub{reply: func(call int) (bool, error) { return call > 1, nil }}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`

	p.send(t, fmt.Sprintf(authenticate, 5))
	p.waitLines(t, 2)
	p.send(t, fmt.Sprintf(authenticate, 6))
	lines := p.waitLines(t, 4)

	assert.Equal(t, []bool{true, false}, signIn.calls())
	assert.Contains(t, lines[2], "Already signed in to system.ai.github")
	require.NoError(t, p.finish(t))
}

func TestAuthenticateReopensPageAfterTimeout(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, listTools))
	signIn := &signInStub{reply: func(call int) (bool, error) {
		if call == 1 {
			return false, &SignInTimeout{URL: testURL}
		}
		return false, nil
	}}
	p := startPiped(t, t.Context(), challengedBridge(srv.URL, signIn.SignIn))
	authenticate := `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`

	p.send(t, fmt.Sprintf(authenticate, 7))
	lines := p.waitLines(t, 1)
	require.Len(t, lines, 1)
	assert.Equal(t, true, parseReply(t, lines[0]).Result["isError"])
	text := resultText(t, lines[0])
	assert.Contains(t, text, "did not complete in 5 minutes. Finish at "+testURL)
	assert.Contains(t, text, "databricks aitools mcp login system.ai.github --profile logfood")

	p.send(t, fmt.Sprintf(authenticate, 8))
	lines = p.waitLines(t, 3)

	assert.Equal(t, []bool{true, true}, signIn.calls(), "a timed-out attempt may open the page again")
	assert.Contains(t, lines[1], "Signed in to system.ai.github")
	require.NoError(t, p.finish(t))
}

func TestAuthenticateReportsFailureWithoutListChanged(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerWith(http.StatusOK, listTools))
	signIn := &signInStub{reply: func(int) (bool, error) { return false, errors.New("sign-in failed for the test") }}
	out, _ := runAll(t, challengedBridge(srv.URL, signIn.SignIn),
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`+"\n")

	require.Len(t, out, 1)
	assert.Equal(t, true, parseReply(t, out[0]).Result["isError"])
	assert.Contains(t, out[0], "Sign-in to system.ai.github on profile logfood failed: sign-in failed for the test")
}

// TestChallengeTurnsOnAfterSignIn runs the flow an agent sees: a challenged list, authenticate, then the
// list passes through once the credential exists upstream.
func TestChallengeTurnsOnAfterSignIn(t *testing.T) {
	var signedIn atomic.Bool
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if signedIn.Load() {
			answerWith(http.StatusOK, listTools)(w, r)
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

func TestRunAdvertisesListChangedOnlyForSignInServices(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answerInitialize)
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n"
	cfg := &config.Config{Host: testHost}

	opts, out, _ := runOptions(t, srv.URL, cfg, initialize)
	require.NoError(t, Run(t.Context(), opts))
	assert.Contains(t, out.String(), `"listChanged":false`)

	opts, out, _ = runOptions(t, srv.URL, cfg, initialize)
	opts.Challenge = &Challenge{Service: testService, Profile: "logfood", Host: testHost, Label: "GitHub"}
	require.NoError(t, Run(t.Context(), opts))
	assert.Contains(t, out.String(), `"listChanged":true`)
}
