package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	authenticateTool = "authenticate"
	signInPath       = "/mcp-service-login"

	codeSignInRequired = -32006
	codeURLElicitation = -32042
)

var listChangedNotification = []byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`)

// SignInTimeout means the sign-in page was shown and the credential did not become ACTIVE within Wait. URL is the page
// the user can finish at. LastCheck is the failure of the last poll, when that poll failed.
type SignInTimeout struct {
	URL       string
	Wait      time.Duration
	LastCheck error
}

func (e *SignInTimeout) Error() string {
	return "sign-in did not complete before the timeout"
}

func (e *SignInTimeout) Unwrap() error {
	return e.LastCheck
}

// SignInRequest says how one sign-in runs.
type SignInRequest struct {
	// Open opens the sign-in page. It is false when the page must not open again in this session.
	Open bool
	// Restart deletes the caller's sign-in before the sign-in runs, so that it starts from no credential. The proxy asks
	// for it when the service challenged the session again after an "Already signed in" answer.
	Restart bool
	// Progress is called on each poll of the wait. It is never nil.
	Progress func()
}

// SignInFunc runs the sign-in for the authenticate tool and waits for it. It reports alreadySignedIn when the caller
// needed no sign-in, and urlShown when the sign-in page's URL was printed, which is so even when the wait is then
// cancelled. It returns a *SignInTimeout when the wait ended first.
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
	// SignIn runs the sign-in. The command provides it, so the proxy never polls or opens a browser itself.
	SignIn SignInFunc
}

// signInHooks applies the sign-in rules to the bridge. Services without a challenge get no hooks.
type signInHooks struct {
	c *Challenge

	// mu serializes authenticate calls, which may run a sign-in for minutes. pageCancelled is read and written under it.
	mu sync.Mutex
	// pageCancelled means the last attempt showed the sign-in page and the client cancelled it before the sign-in
	// completed. The user may still be on that page, so the next attempt does not open it again.
	pageCancelled bool

	// stateMu guards the three fields below. It is held only briefly, so an answer to a request is not held up by a
	// sign-in.
	stateMu sync.Mutex
	// sessionChallenged means a request of this session was challenged. The authenticate tool is the proxy's only after that.
	sessionChallenged bool
	// alreadySignedIn means the last authenticate answered "Already signed in".
	alreadySignedIn bool
	// restart means the next sign-in starts from no credential. A challenge that follows an "Already signed in" answer
	// sets it: the credential reads as signed in, but the service does not accept it.
	restart bool
}

// Local answers the authenticate tool call once the session has been challenged. Before that, and for other requests and
// messages that do not decode, the request is forwarded.
func (h *signInHooks) Local(ctx context.Context, request []byte, notify func([]byte)) ([][]byte, bool) {
	var req struct {
		message
		Params struct {
			Name string `json:"name"`
			Meta struct {
				ProgressToken json.RawMessage `json:"progressToken"`
			} `json:"_meta"`
		} `json:"params"`
	}
	if json.Unmarshal(request, &req) != nil || !req.isRequest() || req.Method != methodToolsCall || req.Params.Name != authenticateTool {
		return nil, false
	}
	h.stateMu.Lock()
	challenged := h.sessionChallenged
	h.stateMu.Unlock()
	if !challenged {
		return nil, false
	}
	return h.authenticate(ctx, req.ID, req.Params.Meta.ProgressToken, notify), true
}

// Result replaces a challenged tools/list answer with the authenticate tool. A challenged tools/call
// answer becomes a sign-in message, followed by tools/list_changed so an agent that cached its list re-lists.
func (h *signInHooks) Result(_ context.Context, request []byte, status int, message []byte) ([][]byte, error) {
	req, err := parseMessage(request)
	if err != nil {
		return nil, err
	}
	switch req.Method {
	case "tools/list":
		if h.isChallenge(status, message) {
			return [][]byte{h.authenticateList(req.ID)}, nil
		}
	case methodToolsCall:
		if h.isChallenge(status, message) {
			return [][]byte{h.signInRequired(req.ID), listChangedNotification}, nil
		}
	}
	return [][]byte{message}, nil
}

// isChallenge reports whether an upstream answer is a challenge, and records that the session has been challenged.
func (h *signInHooks) isChallenge(status int, message []byte) bool {
	if !h.challenged(status, message) {
		return false
	}
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.sessionChallenged = true
	if h.alreadySignedIn {
		h.restart = true
		h.alreadySignedIn = false
	}
	return true
}

// upstreamReply is the part of an upstream message that challenge detection reads.
type upstreamReply struct {
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

// challenged reports whether an upstream answer says the caller must sign in: HTTP 401, JSON-RPC
// -32006, a -32042 whose elicitations are all sign-in URLs, or an isError result whose text has one.
func (h *signInHooks) challenged(status int, message []byte) bool {
	if status == http.StatusUnauthorized {
		return true
	}
	var reply upstreamReply
	if err := json.Unmarshal(message, &reply); err != nil {
		return false
	}
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

// textHasSignInURL reports whether one of the texts has a word that is a sign-in URL. Punctuation around the word, such as
// the brackets of a sentence or the period that ends it, is not part of the URL.
func (h *signInHooks) textHasSignInURL(content []textContent) bool {
	for _, c := range content {
		for word := range strings.FieldsSeq(c.Text) {
			if h.isSignInURL(strings.Trim(word, "([\"'.,;:)]")) {
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
		u.Path == signInPath && strings.EqualFold(u.Query().Get("name"), h.c.Service)
}

// authenticateList is the one-tool list a challenged service answers with.
func (h *signInHooks) authenticateList(id json.RawMessage) []byte {
	tool := map[string]any{
		"name": authenticateTool,
		"description": fmt.Sprintf("Sign-in is required before this Databricks MCP service (%s, workspace %s, CLI profile %s) can list or run its tools. "+
			"Call this tool when the user asks for this service. It opens a browser for the user to sign in to %s with the same account as the Databricks profile "+
			"and waits until the sign-in completes. After it succeeds, retry the user's original request in the same turn.",
			h.c.Service, h.c.Host, h.c.Profile, h.c.Label),
		"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false},
	}
	return resultResponse(id, map[string]any{"tools": []any{tool}})
}

// signInRequired answers a challenged tool call without opening a browser.
func (h *signInHooks) signInRequired(id json.RawMessage) []byte {
	text := fmt.Sprintf("%s on profile %s needs a one-time sign-in to %s. Call the authenticate tool, or run: %s. "+
		"If the consent screen says your organization requires approval, share the app name shown there with your Databricks admin.",
		h.c.Service, h.c.Profile, h.c.Label, h.c.LoginCommand)
	return toolResult(id, text, true)
}

// authenticate runs the sign-in and answers the tool call. Calls are serialized, so a second call waits for the first.
// The page opens unless the last attempt was cancelled while its page was showing (see pageCancelled). With a
// progressToken, each poll of the wait is reported to the client through notify.
func (h *signInHooks) authenticate(ctx context.Context, id, progressToken json.RawMessage, notify func([]byte)) [][]byte {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.stateMu.Lock()
	restart := h.restart
	h.restart = false
	h.stateMu.Unlock()

	progress := func() {}
	if len(progressToken) > 0 {
		polls := 0
		progress = func() {
			polls++
			notify(progressNotification(progressToken, polls, h.c.Label))
		}
	}
	alreadySignedIn, urlShown, err := h.c.SignIn(ctx, SignInRequest{Open: !h.pageCancelled, Restart: restart, Progress: progress})
	h.pageCancelled = urlShown && err != nil && ctx.Err() != nil
	h.stateMu.Lock()
	h.alreadySignedIn = err == nil && alreadySignedIn
	if ctx.Err() != nil {
		// A restart that was cancelled may not have deleted the credential, so the next sign-in restarts again.
		h.restart = h.restart || restart
	}
	h.stateMu.Unlock()
	if ctx.Err() != nil {
		// The client cancelled the request or shutdown began. Nothing is answered for a cancelled request.
		return nil
	}
	timeout, timedOut := errors.AsType[*SignInTimeout](err)

	switch {
	case err == nil && alreadySignedIn:
		// The credential was already there, so nothing changed for the agent's tool list.
		return [][]byte{toolResult(id, fmt.Sprintf("Already signed in to %s on profile %s. Tools are available now; retry the user's request in this turn.", h.c.Service, h.c.Profile), false)}
	case err == nil:
		return [][]byte{
			toolResult(id, fmt.Sprintf("Signed in to %s on profile %s. Tools are available now; retry the user's request in this turn.", h.c.Service, h.c.Profile), false),
			listChangedNotification,
		}
	case timedOut:
		return [][]byte{toolResult(id, fmt.Sprintf(
			"Sign-in to %s did not complete in %g minutes. Finish at %s then retry, or run: %s",
			h.c.Service, timeout.Wait.Minutes(), timeout.URL, h.c.LoginCommand), true)}
	default:
		return [][]byte{toolResult(id, fmt.Sprintf("Sign-in to %s on profile %s failed: %v", h.c.Service, h.c.Profile, err), true)}
	}
}

// progressNotification is the notifications/progress message for the n-th poll of a sign-in that the client asked to
// hear about through token.
func progressNotification(token json.RawMessage, n int, label string) []byte {
	// Cannot fail: the token came from a parsed request and the rest holds plain values.
	b, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/progress",
		"params": map[string]any{
			"progressToken": token,
			"progress":      n,
			"message":       fmt.Sprintf("Waiting for the sign-in to %s to complete.", label),
		},
	})
	return b
}

// resultResponse builds a JSON-RPC success response.
func resultResponse(id json.RawMessage, result any) []byte {
	// Cannot fail: the id came from a parsed request and result holds only plain values.
	b, _ := json.Marshal(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{JSONRPC: "2.0", ID: id, Result: result})
	return b
}

// toolResult builds a tool call result with one text content item.
func toolResult(id json.RawMessage, text string, isError bool) []byte {
	return resultResponse(id, map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
		"isError": isError,
	})
}
