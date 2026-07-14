package testserver

import (
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
	offset     int
	times      int
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
func (fr *FaultRules) Set(token, pattern string, statusCode int, body string, offset, times int) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.rules[faultRuleKey{token: token, pattern: pattern}] = &FaultRule{
		StatusCode: statusCode,
		Body:       body,
		offset:     offset,
		times:      times,
	}
}

// Check returns a matching fault rule and advances its counters, or nil if no rule matches.
// Pattern supports a trailing "*" wildcard, e.g. "PUT /api/2.0/permissions/pipelines/*".
func (fr *FaultRules) Check(method, path, token string) *FaultRule {
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
			Pattern    string `json:"pattern"`
			StatusCode int    `json:"status_code"`
			Body       string `json:"body"`
			Offset     int    `json:"offset"`
			Times      int    `json:"times"`
		}
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return Response{StatusCode: 400, Body: map[string]string{"error": err.Error()}}
		}
		fr.Set(req.Token, body.Pattern, body.StatusCode, body.Body, body.Offset, body.Times)
		return Response{StatusCode: 200}
	}
}
