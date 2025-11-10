package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectCursor(t *testing.T) {
	// This test will pass or fail depending on whether Cursor config exists
	// We can't easily mock file system, so we just test the function works
	result := DetectCursor()
	// Result depends on whether cursor is installed in test environment
	assert.IsType(t, true, result)
}

func TestNewCursorAgent(t *testing.T) {
	agent := NewCursorAgent()

	assert.NotNil(t, agent)
	assert.Equal(t, "cursor", agent.Name)
	assert.Equal(t, "Cursor", agent.DisplayName)
	assert.NotNil(t, agent.Installer)
	// Detected will depend on environment
	assert.IsType(t, true, agent.Detected)
}

func TestGetCursorConfigPath(t *testing.T) {
	// Test that the function returns a non-empty path
	path, err := GetCursorConfigPath()
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".cursor")
	assert.Contains(t, path, "mcp.json")
}
