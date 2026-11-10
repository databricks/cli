// Package agenttest controls which coding agent useragent.AgentProvider()
// reports in tests.
package agenttest

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// SetAgentEnv clears the environment (the agent running the test binary would
// otherwise leak in), then sets the given key/value pairs, and resets the SDK's
// process-wide AgentProvider cache before and after the test. With no
// arguments AgentProvider() reports no agent. Unlike testutil.SetAgent, which
// sets only AGENT, it sets the agents' own env vars (e.g. CLAUDECODE=1), and
// several at once for the "multiple" case. Tests using it must set any other
// env they need (HOME, ...) afterwards and must not run in parallel. A test
// that uses t.TempDir must call it once before SetAgentEnv: the first call
// picks the temp root from TMPDIR / TEMP, which SetAgentEnv clears (on Windows
// the fallback is not writable for a standard user).
func SetAgentEnv(t *testing.T, kv ...string) {
	t.Helper()
	if len(kv)%2 != 0 {
		t.Fatalf("SetAgentEnv needs key/value pairs, got %d values", len(kv))
	}
	testutil.NullEnvironment(t)
	for i := 0; i < len(kv); i += 2 {
		t.Setenv(kv[i], kv[i+1])
	}
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)
}
