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
		location string
		leaf     string
		profile  string
		want     string
	}{
		{name: "system.ai keeps the leaf", location: "system.ai", leaf: "genie_one_mcp", profile: "logfood", want: "genie_one_mcp-logfood"},
		{name: "system.ai github", location: "system.ai", leaf: "github", profile: "dogfood", want: "github-dogfood"},
		{name: "customer schema includes catalog and schema", location: "main.tools", leaf: "crm", profile: "prod", want: "main-tools-crm-prod"},
		{name: "profile is sanitized", location: "system.ai", leaf: "dbsql", profile: "My.Profile", want: "dbsql-my-profile"},
		{name: "system.ai location is case-insensitive", location: "System.AI", leaf: "github", profile: "prod", want: "github-prod"},
		{name: "trailing underscore cannot form __ before the tool name", location: "system.ai", leaf: "github", profile: "prod_", want: "github-prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ServerName(tt.location, tt.leaf, tt.profile))
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
