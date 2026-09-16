package testserver

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
)

type faultRuleKey struct {
	token   string
	pattern string
}

// FaultRule describes a single injected fault: HTTP status, body, and remaining fire count.
type FaultRule struct {
	StatusCode int
	Body       string
	// AfterHandler keeps the handler's state change and replaces only its response.
	AfterHandler bool
	offset       int
	times        int

	// bodyContains, if non-empty, additionally requires the request body to
	// contain this substring for the rule to fire. /workspace/import routes
	// every upload through the same method+path, so the only way to target a
	// single file's upload is by matching its multipart "path" form field.
	bodyContains string
}

// FaultRules holds the active fault injection rules for a test server.
type FaultRules struct {
	mu    sync.Mutex
	rules map[faultRuleKey]*FaultRule
}

// NewFaultRules returns an empty FaultRules.
func NewFaultRules() *FaultRules {
	return &FaultRules{rules: make(map[faultRuleKey]*FaultRule)}
}

// Set registers or replaces a fault rule for the given token and pattern.
func (fr *FaultRules) Set(token, pattern string, statusCode int, body string, offset, times int, bodyContains string) {
	fr.set(token, pattern, statusCode, body, offset, times, false, bodyContains)
}

// SetAfterHandler is like Set, but the handler runs first so its effect is kept.
func (fr *FaultRules) SetAfterHandler(token, pattern string, statusCode int, body string, offset, times int, bodyContains string) {
	fr.set(token, pattern, statusCode, body, offset, times, true, bodyContains)
}

func (fr *FaultRules) set(token, pattern string, statusCode int, body string, offset, times int, afterHandler bool, bodyContains string) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.rules[faultRuleKey{token: token, pattern: pattern}] = &FaultRule{
		StatusCode:   statusCode,
		Body:         body,
		AfterHandler: afterHandler,
		offset:       offset,
		times:        times,
		bodyContains: bodyContains,
	}
}

// Check returns a matching fault rule and advances its counters, or nil if no rule matches.
// Pattern supports a trailing "*" wildcard, e.g. "PUT /api/2.0/permissions/pipelines/*".
// A rule with a non-empty bodyContains only matches when body contains that substring.
func (fr *FaultRules) Check(method, path, token string, body []byte) *FaultRule {
	requestPattern := method + " " + path

	fr.mu.Lock()
	defer fr.mu.Unlock()

	for key, rule := range fr.rules {
		if key.token != token {
			continue
		}
		rulePattern := key.pattern
		var matched bool
		if strings.HasSuffix(rulePattern, "*") {
			matched = strings.HasPrefix(requestPattern, rulePattern[:len(rulePattern)-1])
		} else {
			matched = requestPattern == rulePattern
		}
		if !matched {
			continue
		}
		// A non-matching body must not consume the rule's offset/times budget,
		// so this check precedes the counter updates below.
		if rule.bodyContains != "" && !bytes.Contains(body, []byte(rule.bodyContains)) {
			continue
		}
		if rule.offset > 0 {
			rule.offset--
			return nil
		}
		if rule.times <= 0 {
			delete(fr.rules, key)
			return nil
		}
		rule.times--
		if rule.times == 0 {
			delete(fr.rules, key)
		}
		result := *rule
		return &result
	}

	return nil
}

// faultEndpointHandler handles POST /__testserver/fault.
func faultEndpointHandler(fr *FaultRules) HandlerFunc {
	return func(req Request) any {
		var body struct {
			Pattern      string `json:"pattern"`
			StatusCode   int    `json:"status_code"`
			Body         string `json:"body"`
			Offset       int    `json:"offset"`
			Times        int    `json:"times"`
			AfterHandler bool   `json:"after_handler"`
			BodyContains string `json:"body_contains"`
		}
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return Response{StatusCode: 400, Body: map[string]string{"error": err.Error()}}
		}
		if body.AfterHandler {
			fr.SetAfterHandler(req.Token, body.Pattern, body.StatusCode, body.Body, body.Offset, body.Times, body.BodyContains)
		} else {
			fr.Set(req.Token, body.Pattern, body.StatusCode, body.Body, body.Offset, body.Times, body.BodyContains)
		}
		return Response{StatusCode: 200}
	}
}
