package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "logfood", want: "logfood"},
		{in: "Genie_One", want: "genie_one"},
		{in: "My.Profile", want: "my-profile"},
		{in: "  --a..b--  ", want: "a-b"},
		{in: "..", want: ""},
		{in: "prod_", want: "prod"},
		{in: "my__ws", want: "my_ws"},
		{in: "__a___b__", want: "a_b"},
		{in: "a_.b", want: "a_-b"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, Sanitize(tt.in))
		})
	}
}

func TestServerName(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		profile  string
		want     string
	}{
		{name: "system.ai keeps the leaf", fullName: "system.ai.genie_one_mcp", profile: "logfood", want: "genie_one_mcp-logfood"},
		{name: "system.ai github", fullName: "system.ai.github", profile: "dogfood", want: "github-dogfood"},
		{name: "customer schema includes catalog and schema", fullName: "main.tools.crm", profile: "prod", want: "main-tools-crm-prod"},
		{name: "profile is sanitized", fullName: "system.ai.dbsql", profile: "My.Profile", want: "dbsql-my-profile"},
		{name: "system.ai is case-insensitive", fullName: "System.AI.github", profile: "prod", want: "github-prod"},
		{name: "trailing underscore cannot form __ before the tool name", fullName: "system.ai.github", profile: "prod_", want: "github-prod"},
		{name: "parts that sanitize to nothing are left out", fullName: "main.tools.\u9867\u5ba2", profile: "prod", want: "main-tools-prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ServerName(tt.fullName, tt.profile))
		})
	}
}

func TestUniqueName(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		taken []string
		want  string
	}{
		{name: "free base", base: "github-logfood", want: "github-logfood"},
		{name: "base taken", base: "github-logfood", taken: []string{"github-logfood"}, want: "github-logfood-2"},
		{name: "suffixes taken", base: "github-logfood", taken: []string{"github-logfood", "github-logfood-2"}, want: "github-logfood-3"},
		{name: "suffix only taken", base: "github-logfood", taken: []string{"github-logfood-2"}, want: "github-logfood"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			takenSet := make(map[string]bool)
			for _, name := range tt.taken {
				takenSet[name] = true
			}
			got := UniqueName(tt.base, func(name string) bool { return takenSet[name] })
			assert.Equal(t, tt.want, got)
		})
	}
}
