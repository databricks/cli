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

type faultRule struct {
	statusCode int
	body       string
	offset     int
	times      int
}

type faultRules struct {
	mu    sync.Mutex
	rules map[faultRuleKey]*faultRule
}

func newFaultRules() *faultRules {
	return &faultRules{rules: make(map[faultRuleKey]*faultRule)}
}

func (fr *faultRules) set(token, pattern string, statusCode int, body string, offset, times int) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.rules[faultRuleKey{token: token, pattern: pattern}] = &faultRule{
		statusCode: statusCode,
		body:       body,
		offset:     offset,
		times:      times,
	}
}

// check returns a matching fault rule and advances its counters, or nil if no rule matches.
// Pattern supports a trailing "*" wildcard, e.g. "PUT /api/2.0/permissions/pipelines/*".
func (fr *faultRules) check(method, path, token string) *faultRule {
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
func faultEndpointHandler(fr *faultRules) HandlerFunc {
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
		fr.set(req.Token, body.Pattern, body.StatusCode, body.Body, body.Offset, body.Times)
		return Response{StatusCode: 200}
	}
}
