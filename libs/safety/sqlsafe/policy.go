package sqlsafe

import (
	"slices"
	"strings"
)

type Decision struct {
	Allow          bool
	Reason         string
	BlockedKeyword string
}

type RuleFunc func(p *Policy, stmt Statement, tokens []string) Decision

type Policy struct {
	Keywords map[string]RuleFunc
	Default  RuleFunc
}

// DefaultPolicy returns the conservative policy used by the CLI. Only read-only
// statements are allowed; all other keywords are blocked unless explicitly
// whitelisted.
func DefaultPolicy() Policy {
	p := Policy{
		Keywords: make(map[string]RuleFunc),
	}

	allow := func() RuleFunc {
		return func(_ *Policy, _ Statement, _ []string) Decision {
			return Decision{Allow: true}
		}
	}

	block := func(keyword string) RuleFunc {
		return func(_ *Policy, _ Statement, _ []string) Decision {
			return Decision{
				Allow:          false,
				Reason:         "blocked by read-only policy: keyword " + keyword,
				BlockedKeyword: keyword,
			}
		}
	}

	for _, kw := range []string{"SELECT", "SHOW", "DESCRIBE", "VALUES", "TABLE"} {
		p.Keywords[kw] = allow()
	}

	p.Keywords["WITH"] = withRule(&p)
	p.Keywords["EXPLAIN"] = explainRule(&p)

	blocked := []string{
		"ALTER", "ANALYZE", "CACHE", "CALL", "CLONE", "COMMENT", "COMMIT", "COPY", "CREATE",
		"DELETE", "DETACH", "DROP", "GRANT", "IMPORT", "INSERT", "KILL", "LOAD", "MERGE",
		"MSCK", "OPTIMIZE", "REFRESH", "REPAIR", "REPLACE", "RESET", "RESTORE", "REVOKE",
		"ROLLBACK", "SET", "START", "STOP", "TRUNCATE", "UNCACHE", "UNLOAD", "UPDATE",
		"USE", "VACUUM",
	}
	slices.Sort(blocked)
	for _, kw := range blocked {
		p.Keywords[kw] = block(kw)
	}

	p.Default = func(_ *Policy, _ Statement, tokens []string) Decision {
		keyword := ""
		if len(tokens) > 0 {
			keyword = tokens[0]
		}
		if keyword == "" {
			keyword = "UNKNOWN"
		}
		return Decision{
			Allow:          false,
			Reason:         "blocked by read-only policy: keyword " + keyword,
			BlockedKeyword: keyword,
		}
	}

	return p
}

func withRule(p *Policy) RuleFunc {
	return func(policy *Policy, stmt Statement, tokens []string) Decision {
		keyword, idx := findWithTarget(tokens)
		if keyword == "" {
			return Decision{Allow: false, Reason: "WITH clause does not contain a readable statement", BlockedKeyword: "WITH"}
		}
		decision := policy.decide(stmt, tokens[idx:])
		if !decision.Allow && decision.BlockedKeyword == "" {
			decision.BlockedKeyword = keyword
		}
		return decision
	}
}

func explainRule(p *Policy) RuleFunc {
	return func(policy *Policy, stmt Statement, tokens []string) Decision {
		keyword, idx := findExplainTarget(tokens)
		if keyword == "" {
			return Decision{Allow: false, Reason: "EXPLAIN clause does not contain a readable statement", BlockedKeyword: "EXPLAIN"}
		}
		decision := policy.decide(stmt, tokens[idx:])
		if !decision.Allow && decision.BlockedKeyword == "" {
			decision.BlockedKeyword = keyword
		}
		return decision
	}
}

func (p *Policy) decide(stmt Statement, tokens []string) Decision {
	if len(tokens) == 0 {
		return Decision{Allow: true}
	}
	keyword := strings.ToUpper(tokens[0])
	if rule, ok := p.Keywords[keyword]; ok {
		return rule(p, stmt, tokens)
	}
	if p.Default != nil {
		decision := p.Default(p, stmt, tokens)
		if !decision.Allow && decision.BlockedKeyword == "" {
			decision.BlockedKeyword = keyword
		}
		return decision
	}
	return Decision{Allow: false, Reason: "blocked by read-only policy: keyword " + keyword, BlockedKeyword: keyword}
}

func findWithTarget(tokens []string) (string, int) {
	i := 1
	if i < len(tokens) && tokens[i] == "RECURSIVE" {
		i++
	}
	for i < len(tokens) {
		// Skip CTE name and optional column list.
		for i < len(tokens) && tokens[i] != "AS" {
			if tokens[i] == "(" {
				i = skipParentheses(tokens, i)
				continue
			}
			i++
		}
		if i >= len(tokens) {
			return "", len(tokens)
		}
		// Skip "AS".
		i++
		if i >= len(tokens) {
			return "", len(tokens)
		}
		if i < len(tokens) && tokens[i] == "NOT" {
			i++
		}
		if i < len(tokens) && tokens[i] == "MATERIALIZED" {
			i++
		}
		if i < len(tokens) && tokens[i] == "(" {
			i = skipParentheses(tokens, i)
		}
		if i < len(tokens) && tokens[i] == "," {
			i++
			continue
		}
		break
	}
	if i >= len(tokens) {
		return "", len(tokens)
	}
	return tokens[i], i
}

func findExplainTarget(tokens []string) (string, int) {
	modifiers := map[string]struct{}{
		"ANALYZE": {}, "ANALYZED": {}, "EXTENDED": {}, "FORMATTED": {}, "LOGICAL": {},
		"VERBOSE": {}, "COST": {}, "CODEGEN": {}, "PLAN": {}, "SUMMARY": {},
	}
	i := 1
	for i < len(tokens) {
		tok := tokens[i]
		if tok == "(" {
			i = skipParentheses(tokens, i)
			continue
		}
		if _, ok := modifiers[tok]; ok {
			i++
			continue
		}
		break
	}
	if i >= len(tokens) {
		return "", len(tokens)
	}
	return tokens[i], i
}

func skipParentheses(tokens []string, i int) int {
	depth := 0
	for i < len(tokens) {
		switch tokens[i] {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return len(tokens)
}
