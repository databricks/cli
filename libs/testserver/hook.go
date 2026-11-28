package testserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
)

// hookRule makes the server send a request of its own when a request that matches pattern
// arrives. The request stands for another client that acts at a known point of the test's request
// sequence, e.g. one that creates a resource between a failed create request and its repeat. A
// script can only approximate that with sleeps and background processes.
type hookRule struct {
	pattern *regexp.Regexp
	offset  int
	times   int

	// The request to send, with the token of the request that matched.
	method string
	path   string
	body   string
	// clockOffsetMs shifts the time the handler of this request sees; see Request.nowMilli.
	clockOffsetMs int64
}

type hookRules struct {
	mu    sync.Mutex
	rules map[faultRuleKey]*hookRule
}

func newHookRules() *hookRules {
	return &hookRules{rules: make(map[faultRuleKey]*hookRule)}
}

func (hr *hookRules) set(token, pattern string, rule hookRule) {
	rule.pattern = compileRequestPattern(pattern)
	hr.mu.Lock()
	defer hr.mu.Unlock()
	hr.rules[faultRuleKey{token: token, pattern: pattern}] = &rule
}

// check returns the hook to run for this request and advances its counters, or nil. offset and
// times count as they do for a fault rule.
func (hr *hookRules) check(method, path, token string) *hookRule {
	requestPattern := method + " " + path

	hr.mu.Lock()
	defer hr.mu.Unlock()

	for key, rule := range hr.rules {
		if key.token != token || !rule.pattern.MatchString(requestPattern) {
			continue
		}
		if rule.offset > 0 {
			rule.offset--
			return nil
		}
		rule.times--
		if rule.times <= 0 {
			delete(hr.rules, key)
		}
		result := *rule
		return &result
	}
	return nil
}

type clockOffsetKey struct{}

// sendHookRequest handles the hook's request as if a client had sent it with the given token: it
// is recorded and subject to fault and hook rules like any other request. A hook whose request
// matches its own pattern therefore runs again inside that request while it has times left. Its
// counters were advanced before the request was sent, so it stops when they are used up.
func (s *Server) sendHookRequest(ctx context.Context, token string, hook *hookRule) {
	ctx = context.WithValue(ctx, clockOffsetKey{}, hook.clockOffsetMs)
	req := httptest.NewRequestWithContext(ctx, hook.method, hook.path, strings.NewReader(hook.body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code >= http.StatusBadRequest {
		s.t.Errorf("hook request %s %s failed: %d %s", hook.method, hook.path, rec.Code, rec.Body.String())
	}
}

// hookEndpointHandler handles POST /__testserver/hook.
func hookEndpointHandler(hr *hookRules) HandlerFunc {
	return func(req Request) any {
		var body struct {
			Pattern       string `json:"pattern"`
			Offset        int    `json:"offset"`
			Times         int    `json:"times"`
			Method        string `json:"method"`
			Path          string `json:"path"`
			Body          string `json:"body"`
			ClockOffsetMs int64  `json:"clock_offset_ms"`
		}
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return Response{StatusCode: 400, Body: map[string]string{"error": err.Error()}}
		}
		if body.Method == "" || !strings.HasPrefix(body.Path, "/") || body.Times <= 0 {
			return Response{StatusCode: 400, Body: map[string]string{"error": "a hook needs a method, a path that starts with / and times above 0"}}
		}
		hr.set(req.Token, body.Pattern, hookRule{
			pattern:       nil,
			offset:        body.Offset,
			times:         body.Times,
			method:        body.Method,
			path:          body.Path,
			body:          body.Body,
			clockOffsetMs: body.ClockOffsetMs,
		})
		return Response{StatusCode: 200}
	}
}
