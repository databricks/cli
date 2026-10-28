package fault

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

// Rule describes a single injected fault: HTTP status, body, and remaining fire count.
type Rule struct {
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

// Rules holds the active fault injection rules for a test server.
type Rules struct {
	mu    sync.Mutex
	rules map[faultRuleKey]*Rule
}

// NewRules returns an empty Rules.
func NewRules() *Rules {
	return &Rules{rules: make(map[faultRuleKey]*Rule)}
}

// Set registers or replaces a fault rule for the given token and pattern.
func (fr *Rules) Set(token, pattern string, statusCode int, body string, offset, times int, bodyContains string) {
	fr.set(token, pattern, statusCode, body, offset, times, false, bodyContains)
}

// SetAfterHandler is like Set, but the handler runs first so its effect is kept.
func (fr *Rules) SetAfterHandler(token, pattern string, statusCode int, body string, offset, times int, bodyContains string) {
	fr.set(token, pattern, statusCode, body, offset, times, true, bodyContains)
}

func (fr *Rules) set(token, pattern string, statusCode int, body string, offset, times int, afterHandler bool, bodyContains string) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	fr.rules[faultRuleKey{token: token, pattern: pattern}] = &Rule{
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
func (fr *Rules) Check(method, path, token string, body []byte) *Rule {
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

// Register decodes and registers a rule for token using the fault.py wire format.
func (fr *Rules) Register(token string, data []byte) error {
	var body struct {
		Pattern      string `json:"pattern"`
		StatusCode   int    `json:"status_code"`
		Body         string `json:"body"`
		Offset       int    `json:"offset"`
		Times        int    `json:"times"`
		AfterHandler bool   `json:"after_handler"`
		BodyContains string `json:"body_contains"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	if body.AfterHandler {
		fr.SetAfterHandler(token, body.Pattern, body.StatusCode, body.Body, body.Offset, body.Times, body.BodyContains)
	} else {
		fr.Set(token, body.Pattern, body.StatusCode, body.Body, body.Offset, body.Times, body.BodyContains)
	}
	return nil
}
