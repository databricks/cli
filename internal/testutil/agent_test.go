package testutil_test

import (
	"os"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestSetAgentEnv(t *testing.T) {
	kept := t.TempDir()
	t.Setenv("TMPDIR", kept)
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")
	assert.Equal(t, "claude-code", useragent.AgentProvider())
	assert.Equal(t, kept, os.Getenv("TMPDIR"))
}

func TestSetAgentEnvWithoutPairs(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	testutil.SetAgentEnv(t)
	assert.Empty(t, useragent.AgentProvider())
}

func TestSetAgent(t *testing.T) {
	testutil.SetAgent(t, "my-agent")
	assert.Equal(t, "my-agent", useragent.AgentProvider())
}
