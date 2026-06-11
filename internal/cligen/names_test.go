package main

import "testing"

// TestNameCasings guards the ported genkit name functions (names.go). The
// producer no longer denormalizes casings into cliv1.json; cligen derives them
// from the stored name, so these must match genkit exactly. The values below
// were validated against genkit's output for all named objects in cliv1.json
// before the stored casings were dropped.
func TestNameCasings(t *testing.T) {
	cases := []struct {
		name, kebab, snake, pascal, camel, constant, title string
	}{
		{"JobSettings", "job-settings", "job_settings", "JobSettings", "jobSettings", "JOB_SETTINGS", "Job Settings"},
		{"Workspace", "workspace", "workspace", "Workspace", "workspace", "WORKSPACE", "Workspace"},
		{"IpAccessLists", "ip-access-lists", "ip_access_lists", "IpAccessLists", "ipAccessLists", "IP_ACCESS_LISTS", "Ip Access Lists"},
		{"IamV2", "iam-v2", "iam_v2", "IamV2", "iamV2", "IAM_V2", "Iam V2"},
		{"create_run", "create-run", "create_run", "CreateRun", "createRun", "CREATE_RUN", "Create Run"},
		// Digit-then-letter: digits are not word separators (strings.Title
		// semantics), so the letter after a digit stays lowercase. The Go SDK
		// spells these fields the same way (e.g. serving.Ai21labsApiKey).
		{"ai21labs_api_key", "ai21labs-api-key", "ai21labs_api_key", "Ai21labsApiKey", "ai21labsApiKey", "AI21LABS_API_KEY", "Ai21labs Api Key"},
		{"oauth2client", "oauth2client", "oauth2client", "Oauth2client", "oauth2client", "OAUTH2CLIENT", "Oauth2client"},
		// Consecutive-uppercase acronym: splitASCII splits before the last
		// upper of a run when a lowercase follows.
		{"HTMLParser", "html-parser", "html_parser", "HtmlParser", "htmlParser", "HTML_PARSER", "Html Parser"},
		// Empty and "_" hit the special-cased early returns in camelName/snakeName;
		// note kebab/title/pascal do not special-case them, so they differ.
		{"", "", "", "", "", "", ""},
		{"_", "", "_", "", "_", "_", ""},
	}
	for _, c := range cases {
		if got := kebabName(c.name); got != c.kebab {
			t.Errorf("kebabName(%q) = %q, want %q", c.name, got, c.kebab)
		}
		if got := snakeName(c.name); got != c.snake {
			t.Errorf("snakeName(%q) = %q, want %q", c.name, got, c.snake)
		}
		if got := pascalName(c.name); got != c.pascal {
			t.Errorf("pascalName(%q) = %q, want %q", c.name, got, c.pascal)
		}
		if got := camelName(c.name); got != c.camel {
			t.Errorf("camelName(%q) = %q, want %q", c.name, got, c.camel)
		}
		if got := constantName(c.name); got != c.constant {
			t.Errorf("constantName(%q) = %q, want %q", c.name, got, c.constant)
		}
		if got := titleName(c.name); got != c.title {
			t.Errorf("titleName(%q) = %q, want %q", c.name, got, c.title)
		}
	}
}

// TestTrimPrefix guards Named.TrimPrefix("account") used for account services.
func TestTrimPrefix(t *testing.T) {
	cases := []struct{ name, want string }{
		{"AccountMetastoreAssignments", "metastore-assignments"},
		{"AccountStorageCredentials", "storage-credentials"},
		{"Workspaces", "workspaces"}, // no "account" prefix: unchanged
	}
	for _, c := range cases {
		tn := trimPrefixName(c.name, "account")
		if got := kebabName(tn); got != c.want {
			t.Errorf("kebabName(trimPrefix(%q,\"account\")) = %q, want %q", c.name, got, c.want)
		}
	}
}
