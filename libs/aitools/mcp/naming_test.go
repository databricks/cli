package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serverName returns ServerName for a name that must have one.
func serverName(t *testing.T, fullName, profile string) string {
	t.Helper()
	name, err := ServerName(fullName, profile)
	require.NoError(t, err)
	return name
}

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
		{name: "catalog or schema that sanitize to nothing are left out", fullName: "main.\u9867\u5ba2.crm", profile: "prod", want: "main-crm-prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, serverName(t, tt.fullName, tt.profile))
		})
	}
}

func TestServerNameLeafWithoutServerNameCharacters(t *testing.T) {
	for _, fullName := range []string{"main.tools.顧客", "system.ai.顧客", "main.tools.---"} {
		t.Run(fullName, func(t *testing.T) {
			name, err := ServerName(fullName, "prod")
			assert.ErrorIs(t, err, ErrNoServerName)
			assert.ErrorContains(t, err, "has no characters that can be used in a server name")
			assert.Empty(t, name)
		})
	}
}

func TestServerNameCap(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		profile  string
		want     string
	}{
		{name: "32 characters is kept", fullName: "main.sales.abcdefghijklmnop", profile: "prod", want: "main-sales-abcdefghijklmnop-prod"},
		{name: "33 characters is cut and hashed", fullName: "main.sales.abcdefghijklmnopq", profile: "prod", want: "main-sales-abcdefghijklmn-192311"},
		{name: "cut that ends on - drops it", fullName: "main.sales.abcdefghijklm", profile: "production", want: "main-sales-abcdefghijklm-04de7a"},
		{name: "cut that ends on _ drops it", fullName: "main.sales.abcdefghijklm_n", profile: "production", want: "main-sales-abcdefghijklm-6d4c3a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, serverName(t, tt.fullName, tt.profile))
		})
	}
}

func TestServerNameCapKeepsSharedPrefixesDistinct(t *testing.T) {
	long := serverName(t, "main.sales.abcdefghijklm", "production")
	underscored := serverName(t, "main.sales.abcdefghijklm_n", "production")
	assert.NotEqual(t, long, underscored)
	assert.Equal(t, long, serverName(t, "main.sales.abcdefghijklm", "production"))
}

func TestServerNameCapFollowsSanitizeRules(t *testing.T) {
	for _, name := range []string{
		serverName(t, "main.sales.abcdefghijklm_n", "production"),
		serverName(t, "main.sales.abcdefghijklm-n", "production"),
		serverName(t, "main.sales.abcdefghijklmnopqrstuvwxyz", "production_"),
	} {
		assert.LessOrEqual(t, len(name), MaxServerNameLen, name)
		assert.NotContains(t, name, "__", name)
		assert.NotRegexp(t, `^[-_]|[-_]$`, name)
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

func TestUniqueNameSuffixStaysWithinCap(t *testing.T) {
	base := "main-sales-abcdefghijklmnop-prod"
	require.Len(t, base, MaxServerNameLen)
	taken := func(name string) bool { return name == base }
	got := UniqueName(base, taken)
	assert.LessOrEqual(t, len(got), MaxServerNameLen, got)
	assert.NotEqual(t, base, got)
	assert.Equal(t, got, UniqueName(base, taken))
}
