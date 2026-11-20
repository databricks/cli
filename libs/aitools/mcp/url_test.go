package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceURL(t *testing.T) {
	assert.Equal(t,
		"https://adb-123.azuredatabricks.test/ai-gateway/mcp-services/system.ai.github",
		ServiceURL("https://adb-123.azuredatabricks.test", "system.ai.github"))
}

func TestServiceURLEscapesTheName(t *testing.T) {
	assert.Equal(t,
		"https://adb-123.azuredatabricks.test/ai-gateway/mcp-services/main.tools.crm%23v2",
		ServiceURL("https://adb-123.azuredatabricks.test", "main.tools.crm#v2"))
}

func TestSignInURL(t *testing.T) {
	got, err := SignInURL("https://adb-123.azuredatabricks.test", "system.ai.github", "2548836972759138")
	assert.NoError(t, err)
	assert.Equal(t,
		"https://adb-123.azuredatabricks.test/mcp-service-login?name=system.ai.github&o=2548836972759138",
		got)
}

func TestSignInURLPassesNonNumericWorkspaceAsW(t *testing.T) {
	got, err := SignInURL("https://adb-123.azuredatabricks.test", "system.ai.github", "ws-alias")
	assert.NoError(t, err)
	assert.Equal(t,
		"https://adb-123.azuredatabricks.test/mcp-service-login?name=system.ai.github&w=ws-alias",
		got)
}

func TestSignInURLRefusesUnknownWorkspace(t *testing.T) {
	for _, workspaceID := range []string{"", "none"} {
		t.Run(workspaceID, func(t *testing.T) {
			got, err := SignInURL("https://adb-123.azuredatabricks.test", "system.ai.github", workspaceID)
			assert.ErrorContains(t, err, "the workspace ID is not known")
			assert.Empty(t, got)
		})
	}
}
