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
	assert.Equal(t,
		"https://adb-123.azuredatabricks.test/mcp-service-login?name=system.ai.github&o=2548836972759138",
		SignInURL("https://adb-123.azuredatabricks.test", "system.ai.github", "2548836972759138"))
}
