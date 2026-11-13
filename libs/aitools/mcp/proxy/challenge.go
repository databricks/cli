package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/aitools/mcp"
)

const (
	authenticateTool = "authenticate"

	codeSignInRequired = -32006
	codeURLElicitation = -32042
)

// urlPattern finds the http and https URLs in a text. A URL ends at a space or at a character that cannot be in one
// here, such as a bracket, a quote or an emphasis mark, so the URL of a markdown link or a `<url>` is found too.
var urlPattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile("https?://[^\\s<>()\\[\\]\"'*`]+")
})

// queueProgressInterval is how often a call that waits for the sign-in ahead of it reports progress. Tests shorten it.
var queueProgressInterval = 2 * time.Second

var listChangedNotification = []byte(`{"jsonrpc":"` + jsonrpcVersion + `","method":"` + methodListChanged + `"}`)

// SignInRequest says how one sign-in runs.
type SignInRequest struct {
	// Open opens the sign-in page. It is false when the page must not open again in this session.
	Open bool
	// Restart deletes the caller's sign-in before the sign-in runs, so that it starts from no credential. The proxy asks
	// for it when the service challenged the session again after an "Already signed in" answer.
	Restart bool
	// Progress is called on each poll of the wait with the sign-in page, so that the client can show where the user
	// finishes, whether or not this attempt opened the page. It is never nil.
	Progress func(pageURL string)
	// Stderr receives the progress lines of the sign-in. Writes to it are serialized with the bridge's diagnostics.
	Stderr io.Writer
}

// SignInFunc runs the sign-in for the authenticate tool and waits for it. It reports alreadySignedIn when the caller
// needed no sign-in, and urlShown when the sign-in page's URL was printed, which is so even when the wait is then
// cancelled. Its error is the answer's text.
type SignInFunc func(ctx context.Context, req SignInRequest) (alreadySignedIn, urlShown bool, err error)

// Challenge turns on sign-in handling for a service that needs a provider sign-in.
type Challenge struct {
	// Service is the full name of the MCP service, for example system.ai.github.
	Service string
	// Profile is the bound profile's name.
	Profile string
	// Host is the bound profile's workspace URL. Sign-in URLs must be on this host.
	Host string
	// Label names the provider in messages.
	Label string
	// LoginCommand is the command that signs the profile in to the service, as the hints print it.
	LoginCommand string
	// PinnedFile is the config file the server is pinned to, or "" (see mcp.PinnedFile). The auth hint names it.
	PinnedFile string
	// SignIn runs the sign-in. The command provides it, so the proxy never polls or opens a browser itself. Its error is
	// the answer's text, so it names the service, the profile and the fix.
	SignIn SignInFunc
}

// signInHooks applies the sign-in rules to the bridge. Services without a challenge get no hooks.
type signInHooks struct {
	c *Challenge
	// stderr is the bridge's stderr, which SignIn writes its progress to.
	stderr io.Writer

	// turn is held by the authenticate call whose sign-in runs, so that the calls run one at a time. It has room for one
	// call. A call waits for it under its own context (see acquire). pageCancelled is read and written under it.
	turn chan struct{}
	// pageCancelled means the last attempt showed the sign-in page and the client cancelled it before the sign-in
	// completed. The user may still be on that page, so the next attempt does not open it again.
	pageCancelled bool

	// stateMu guards the three fields below. It is held only briefly, so an answer to a request is not held up by a
	// sign-in.
	stateMu sync.Mutex
	// exposed means the agent's list shows the authenticate tool: a challenge sets it, and a tools/list that passes through
	// unchallenged clears it. The tool is the proxy's only while it is exposed.
	exposed bool
	// alreadySignedIn means the last authenticate answered "Already signed in".
	alreadySignedIn bool
	// restart means the next sign-in starts from no credential. A challenge that follows an "Already signed in" answer
	// sets it: the credential reads as signed in, but the service does not accept it.
	restart bool
	// callChallenged means the last challenge came from a tools/call, and no sign-in has ended and no call has passed
	// since. The agent was told to call authenticate, so a tools/list that passes still lists it.
	callChallenged bool
}

// newSignInHooks returns the sign-in hooks of c. Progress lines of a sign-in are written to stderr.
func newSignInHooks(c *Challenge, stderr io.Writer) *signInHooks {
	return &signInHooks{c: c, stderr: stderr, turn: make(chan struct{}, 1)}
}

// Local answers the authenticate tool call once the session has been challenged. Before that, and for other requests and
// messages that do not decode, the request is forwarded.
func (h *signInHooks) Local(ctx context.Context, request []byte, notify func([]byte)) ([][]byte, bool) {
	var req struct {
		message
		Params struct {
			Name string `json:"name"`
			Meta struct {
				ProgressToken *json.RawMessage `json:"progressToken"`
			} `json:"_meta"`
		} `json:"params"`
	}
	if json.Unmarshal(request, &req) != nil || !req.isRequest() || req.Method != methodToolsCall || req.Params.Name != authenticateTool {
		return nil, false
	}
	h.stateMu.Lock()
	exposed := h.exposed
	h.stateMu.Unlock()
	if !exposed {
		return nil, false
	}
	var progressToken json.RawMessage
	if req.Params.Meta.ProgressToken != nil {
		progressToken = *req.Params.Meta.ProgressToken
	}
	return h.authenticate(ctx, req.ID, progressToken, notify), true
}

// Result replaces a challenged tools/list answer with the authenticate tool. A challenged tools/call
// answer becomes a sign-in message, followed by tools/list_changed so an agent that cached its list re-lists. A 401 that
// is not a challenge becomes the auth failure (see authFailure). Only the response to the request is read; any other
// message passes through untouched.
func (h *signInHooks) Result(_ context.Context, request []byte, status int, message []byte) ([][]byte, error) {
	req, err := parseMessage(request)
	if err != nil {
		return nil, err
	}
	var reply upstreamReply
	if err := json.Unmarshal(message, &reply); err != nil {
		return nil, err
	}
	if !reply.isResponseTo(req) {
		return [][]byte{message}, nil
	}
	challenged := h.challenged(&reply)
	if status == http.StatusUnauthorized && !challenged {
		return [][]byte{errorResponse(req.ID, codeUpstream, h.authFailure())}, nil
	}
	if req.Method != methodToolsList && req.Method != methodToolsCall {
		return [][]byte{message}, nil
	}
	if challenged {
		h.recordChallenge(req.Method)
		if req.Method == methodToolsList {
			return [][]byte{h.authenticateList(req.ID)}, nil
		}
		return [][]byte{h.signInRequired(req.ID), listChangedNotification}, nil
	}
	// A result with no error means the service accepts the caller's credential. An error answer, such as a JSON-RPC
	// error on HTTP 200 or a failure the bridge wrote for the request, says nothing about it.
	if status/100 == 2 && reply.Error == nil && reply.Result != nil {
		msg, err := h.succeeded(req.Method, message)
		if err != nil {
			return nil, err
		}
		return [][]byte{msg}, nil
	}
	return [][]byte{message}, nil
}

// recordChallenge records a challenge to a request of method. The authenticate tool is listed from now on, and a challenge
// that follows an "Already signed in" answer asks for a restart of the sign-in.
func (h *signInHooks) recordChallenge(method string) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.exposed = true
	h.callChallenged = method == methodToolsCall
	if h.alreadySignedIn {
		h.restart = true
		h.alreadySignedIn = false
	}
}

// succeeded records a successful answer to a request of method, and returns the message to write. The "Already signed in"
// state and a pending restart end: a challenge after this answer is not a challenge to the credential that was reported
// signed in, and the service accepts the credential, so there is nothing to restart. A successful tools/call ends a call
// challenge. A tools/list answer takes the authenticate tool off the list, since the list is the service's again, unless
// a call challenge is still pending; then the tool stays on it.
func (h *signInHooks) succeeded(method string, message []byte) ([]byte, error) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.alreadySignedIn = false
	h.restart = false
	if method == methodToolsCall {
		h.callChallenged = false
		return message, nil
	}
	if !h.callChallenged {
		h.exposed = false
		return message, nil
	}
	return h.withAuthenticate(message)
}

// withAuthenticate adds the authenticate tool to the tools of a tools/list result, which has a result.
func (h *signInHooks) withAuthenticate(msg []byte) ([]byte, error) {
	var env map[string]json.RawMessage
	if err := json.Unmarshal(msg, &env); err != nil {
		return nil, err
	}
	result, err := decodeObject(env[keyResult])
	if err != nil {
		return nil, fmt.Errorf("tools/list result: %w", err)
	}
	var tools []json.RawMessage
	if err := json.Unmarshal(member(result, keyTools), &tools); err != nil {
		return nil, fmt.Errorf("tools/list tools: %w", err)
	}
	tool, err := json.Marshal(h.authenticateToolDef())
	if err != nil {
		return nil, err
	}
	if result[keyTools], err = json.Marshal(append(tools, tool)); err != nil {
		return nil, err
	}
	if env[keyResult], err = json.Marshal(result); err != nil {
		return nil, err
	}
	return json.Marshal(env)
}

// upstreamReply is an upstream message as challenge detection reads it: its envelope, and its error or result.
type upstreamReply struct {
	message
	Error *struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	} `json:"error"`
	Result *struct {
		IsError bool          `json:"isError"`
		Content []textContent `json:"content"`
	} `json:"result"`
}

type textContent struct {
	Text string `json:"text"`
}

// challenged reports whether an upstream answer says the caller must sign in: JSON-RPC -32006, a -32042 whose elicitations
// are all sign-in URLs, or an isError result whose text has one. An HTTP 401 is a challenge only with one of these bodies.
func (h *signInHooks) challenged(reply *upstreamReply) bool {
	if reply.Error != nil {
		switch reply.Error.Code {
		case codeSignInRequired:
			return true
		case codeURLElicitation:
			return h.urlElicitation(reply.Error.Data)
		}
		return false
	}
	return reply.Result != nil && reply.Result.IsError && h.textHasSignInURL(reply.Result.Content)
}

// urlElicitation reports whether data lists one or more URL elicitations, all of them sign-in URLs.
// A mixed challenge, such as an approval request, is not a sign-in.
func (h *signInHooks) urlElicitation(data json.RawMessage) bool {
	var d struct {
		Elicitations []struct {
			Mode string `json:"mode"`
			URL  string `json:"url"`
		} `json:"elicitations"`
	}
	if json.Unmarshal(data, &d) != nil || len(d.Elicitations) == 0 {
		return false
	}
	for _, e := range d.Elicitations {
		if e.Mode != "url" || !h.isSignInURL(e.URL) {
			return false
		}
	}
	return true
}

// textHasSignInURL reports whether one of the texts contains a sign-in URL. A period, comma or colon after the URL is
// the punctuation of the sentence, not part of the URL.
func (h *signInHooks) textHasSignInURL(content []textContent) bool {
	for _, c := range content {
		for _, found := range urlPattern().FindAllString(c.Text, -1) {
			if h.isSignInURL(strings.TrimRight(found, ".,:;")) {
				return true
			}
		}
	}
	return false
}

// isSignInURL reports whether raw is this service's sign-in page on the profile's host.
func (h *signInHooks) isSignInURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	base, err := url.Parse(h.c.Host)
	if err != nil {
		return false
	}
	return u.Scheme == base.Scheme && strings.EqualFold(u.Host, base.Host) &&
		u.Path == mcp.SignInPath && strings.EqualFold(u.Query().Get("name"), h.c.Service)
}

// authFailure is the answer to a 401 that does not ask for a sign-in: the Databricks credential was rejected, and the
// profile's login fixes that. The text is the refresh guard's, so the two read alike.
func (h *signInHooks) authFailure() string {
	cause := errors.New("the MCP service rejected the credential of the profile (HTTP 401)")
	return fmt.Sprintf("%v, then restart the MCP server in your agent", mcp.AuthError(h.c.Profile, h.c.PinnedFile, cause))
}

// authenticateList is the one-tool list a challenged service answers with.
func (h *signInHooks) authenticateList(id json.RawMessage) []byte {
	return resultResponse(id, map[string]any{"tools": []any{h.authenticateToolDef()}})
}

// authenticateToolDef is the definition of the authenticate tool, as a tools/list result lists it.
func (h *signInHooks) authenticateToolDef() map[string]any {
	return map[string]any{
		"name": authenticateTool,
		"description": fmt.Sprintf("Sign-in is required before this Databricks MCP service (%s, workspace %s, CLI profile %s) can list or run its tools. "+
			"Call this tool when the user asks for this service. It opens a browser for the user to sign in to %s with the same account as the Databricks profile "+
			"and waits until the sign-in completes. After it succeeds, retry the user's original request in the same turn.",
			h.c.Service, h.c.Host, h.c.Profile, h.c.Label),
		"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false},
	}
}

// signInRequired answers a challenged tool call without opening a browser.
func (h *signInHooks) signInRequired(id json.RawMessage) []byte {
	text := fmt.Sprintf("%s on profile %s needs a one-time sign-in to %s. Call the authenticate tool, or run: %s. "+
		"If the consent screen says your organization requires approval, share the app name shown there with your Databricks admin.",
		h.c.Service, h.c.Profile, h.c.Label, h.c.LoginCommand)
	return toolResult(id, text, true)
}

// authenticate runs the sign-in and answers the tool call. The page opens unless the last attempt was cancelled while its
// page was showing (see pageCancelled). With a progressToken, each poll is reported through notify.
func (h *signInHooks) authenticate(ctx context.Context, id, progressToken json.RawMessage, notify func([]byte)) [][]byte {
	var polls int
	report := func(pageURL string) {
		// A poll that fires after the client cancelled the call reports nothing for it.
		if len(progressToken) == 0 || ctx.Err() != nil {
			return
		}
		polls++
		notify(progressNotification(progressToken, polls, h.c.Label, pageURL))
	}
	if !h.acquire(ctx, report) {
		return nil
	}
	defer h.release()

	h.stateMu.Lock()
	restart := h.restart
	h.restart = false
	h.stateMu.Unlock()

	alreadySignedIn, urlShown, err := h.c.SignIn(ctx, SignInRequest{Open: !h.pageCancelled, Restart: restart, Progress: report, Stderr: h.stderr})
	h.pageCancelled = urlShown && err != nil && ctx.Err() != nil
	h.stateMu.Lock()
	h.alreadySignedIn = err == nil && alreadySignedIn
	if err == nil {
		// The sign-in ended, so the next list is the service's own, unless it is challenged again.
		h.callChallenged = false
	}
	if restart && !urlShown && err != nil {
		// A restart that failed before its page was shown may not have deleted the credential, so it is asked for again.
		h.restart = true
	}
	h.stateMu.Unlock()
	if ctx.Err() != nil {
		// A cancelled request is not answered.
		return nil
	}
	switch {
	case err == nil && alreadySignedIn:
		// The credential was there already, for example from `mcp login` in a terminal, while the agent's list still
		// shows only authenticate, so the agent lists again.
		return [][]byte{
			toolResult(id, fmt.Sprintf("Already signed in to %s on profile %s. Tools are available now; retry the user's request in this turn.", h.c.Service, h.c.Profile), false),
			listChangedNotification,
		}
	case err == nil:
		return [][]byte{
			toolResult(id, fmt.Sprintf("Signed in to %s on profile %s. Tools are available now; retry the user's request in this turn.", h.c.Service, h.c.Profile), false),
			listChangedNotification,
		}
	default:
		return [][]byte{toolResult(id, err.Error(), true)}
	}
}

// acquire takes the turn that lets an authenticate call run its sign-in. While the call waits for the turn, it reports
// progress every queueProgressInterval, so that a client does not give up on it, and it returns false when ctx is done
// first. A call whose ctx is done once it has the turn gives the turn back and returns false, so a call the client
// cancelled before its own attempt starts touches no state.
func (h *signInHooks) acquire(ctx context.Context, report func(pageURL string)) bool {
	ticker := time.NewTicker(queueProgressInterval)
	defer ticker.Stop()
	for {
		select {
		case h.turn <- struct{}{}:
			if ctx.Err() != nil {
				h.release()
				return false
			}
			return true
		case <-ticker.C:
			report("")
		case <-ctx.Done():
			return false
		}
	}
}

// release gives the turn taken by acquire to the next call.
func (h *signInHooks) release() {
	<-h.turn
}

// progressNotification is the notifications/progress message for the n-th poll of a sign-in that the client asked to
// hear about through token. A non-empty pageURL is the page the user finishes at.
func progressNotification(token json.RawMessage, n int, label, pageURL string) []byte {
	message := fmt.Sprintf("Waiting for the sign-in to %s to complete.", label)
	if pageURL != "" {
		message += " Finish at " + pageURL
	}
	// Cannot fail: the token came from a parsed request and the rest holds plain values.
	b, _ := json.Marshal(map[string]any{
		"jsonrpc": jsonrpcVersion,
		"method":  methodProgress,
		"params": map[string]any{
			"progressToken": token,
			"progress":      n,
			"message":       message,
		},
	})
	return b
}

// toolResult builds a tool call result with one text content item.
func toolResult(id json.RawMessage, text string, isError bool) []byte {
	return resultResponse(id, map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
		"isError": isError,
	})
}
