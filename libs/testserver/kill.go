package testserver

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/databricks/cli/internal/testutil"
)

type killRuleKey struct {
	token   string
	pattern string // "METHOD /path"
}

type killRule struct {
	offset int
	times  int
}

type killRules struct {
	mu    sync.Mutex
	rules map[killRuleKey]*killRule
}

func newKillRules() *killRules {
	return &killRules{rules: make(map[killRuleKey]*killRule)}
}

func (kr *killRules) set(token, pattern string, offset, times int) {
	kr.mu.Lock()
	defer kr.mu.Unlock()
	kr.rules[killRuleKey{token: token, pattern: pattern}] = &killRule{offset: offset, times: times}
}

// check returns true if the caller should be killed for this request.
// It also performs the kill.
func (kr *killRules) check(t testutil.TestingT, method, path, token string, headers http.Header) bool {
	pattern := method + " " + path
	key := killRuleKey{token: token, pattern: pattern}

	kr.mu.Lock()
	rule, ok := kr.rules[key]
	if !ok {
		kr.mu.Unlock()
		return false
	}
	if rule.offset > 0 {
		rule.offset--
		kr.mu.Unlock()
		return false
	}
	if rule.times <= 0 {
		delete(kr.rules, key)
		kr.mu.Unlock()
		return false
	}
	rule.times--
	if rule.times == 0 {
		delete(kr.rules, key)
	}
	kr.mu.Unlock()

	killProcess(t, pattern, headers)
	return true
}

func killProcess(t testutil.TestingT, pattern string, headers http.Header) {
	pid := ExtractPidFromHeaders(headers)
	if pid == 0 {
		t.Errorf("kill rule matched %q but test-pid not found in User-Agent", pattern)
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		t.Errorf("Failed to find process %d: %s", pid, err)
		return
	}

	if err := process.Kill(); err != nil {
		t.Errorf("Failed to kill process %d: %s", pid, err)
		return
	}

	if !waitForProcessExit(pid, 2*time.Second) {
		t.Logf("kill: timed out waiting for PID %d to exit (pattern: %s)", pid, pattern)
	}
	t.Logf("kill: killed PID %d (pattern: %s)", pid, pattern)
}

// killEndpointHandler returns a HandlerFunc for POST /__testserver/kill.
func killEndpointHandler(kr *killRules) HandlerFunc {
	return func(req Request) any {
		var body struct {
			Pattern string `json:"pattern"`
			Offset  int    `json:"offset"`
			Times   int    `json:"times"`
		}
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return Response{StatusCode: 400, Body: map[string]string{"error": err.Error()}}
		}
		kr.set(req.Token, body.Pattern, body.Offset, body.Times)
		return Response{StatusCode: 200}
	}
}
