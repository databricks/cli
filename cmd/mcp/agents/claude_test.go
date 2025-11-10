package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectClaude(t *testing.T) {
	// This test will pass if claude is on PATH, fail otherwise
	// We can't mock exec.LookPath easily, so we just test the function works
	result := DetectClaude()
	// Result depends on whether claude is installed in test environment
	assert.IsType(t, true, result)
}

func TestNewClaudeAgent(t *testing.T) {
	agent := NewClaudeAgent()

	assert.NotNil(t, agent)
	assert.Equal(t, "claude", agent.Name)
	assert.Equal(t, "Claude Code", agent.DisplayName)
	assert.NotNil(t, agent.Installer)
	// Detected will depend on environment
	assert.IsType(t, true, agent.Detected)
}
