package u2m

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
)

func TestClientIDFromProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *profile.Profile
		want    string
	}{
		{name: "no profile"},
		{
			name:    "implicit auth type",
			profile: &profile.Profile{ClientID: "custom-client-id"},
		},
		{
			name:    "M2M auth type",
			profile: &profile.Profile{AuthType: "oauth-m2m", ClientID: "custom-client-id"},
		},
		{
			name:    "U2M auth type",
			profile: &profile.Profile{AuthType: AuthTypeDatabricksCLI, ClientID: "custom-client-id"},
			want:    "custom-client-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ClientIDFromProfile(tt.profile))
		})
	}
}

func TestResourcesFromProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile *profile.Profile
		want    []string
	}{
		{name: "no profile"},
		{
			name:    "implicit auth type",
			profile: &profile.Profile{Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github"},
		},
		{
			name:    "M2M auth type",
			profile: &profile.Profile{AuthType: "oauth-m2m", Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github"},
		},
		{
			name:    "U2M auth type single resource",
			profile: &profile.Profile{AuthType: AuthTypeDatabricksCLI, Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github"},
			want:    []string{"https://workspace.test/ai-gateway/mcp/system.ai.github"},
		},
		{
			name:    "U2M auth type multiple resources",
			profile: &profile.Profile{AuthType: AuthTypeDatabricksCLI, Resources: "https://workspace.test/ai-gateway/mcp/system.ai.github, https://workspace.test/ai-gateway/mcp/system.ai.slack"},
			want:    []string{"https://workspace.test/ai-gateway/mcp/system.ai.github", "https://workspace.test/ai-gateway/mcp/system.ai.slack"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ResourcesFromProfile(tt.profile))
		})
	}
}
