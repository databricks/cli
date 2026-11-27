//go:build unix

package installer

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAgentCmdTimeoutStopsDescendants(t *testing.T) {
	// The agent CLI leaves a child holding its output pipes. Killing only the
	// CLI would leave the command waiting for the child to exit.
	start := time.Now()
	_, err := runAgentCmd(t.Context(), 100*time.Millisecond, []string{"sh", "-c", "sleep 30 & sleep 30"})
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second)
	_, timedOut := errors.AsType[*agentCmdTimeoutError](err)
	assert.True(t, timedOut)
}
