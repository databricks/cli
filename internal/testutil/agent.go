package testutil

import (
	"os"

	"github.com/databricks/databricks-sdk-go/useragent"
)

// SetAgent makes useragent.AgentProvider report agent, or no agent if agent
// is empty. It clears the environment and the SDK's cached agent, so the agent
// running the tests does not leak into them.
// Note: use of this function is incompatible with parallel execution.
func SetAgent(t TestingT, agent string) {
	SetAgentEnv(t, "AGENT", agent)
}

// SetAgentEnv clears the environment and sets the given key/value pairs, e.g.
// the agents' own env vars (CLAUDECODE=1), so useragent.AgentProvider sees
// only them; with no pairs it reports no agent. TMPDIR, TEMP and TMP are kept
// so t.TempDir keeps working. The SDK's cached agent is reset before and after
// the test.
// Note: use of this function is incompatible with parallel execution.
func SetAgentEnv(t TestingT, kv ...string) {
	t.Helper()
	if len(kv)%2 != 0 {
		t.Fatalf("SetAgentEnv needs key/value pairs, got %d values", len(kv))
	}
	var keep []string
	for _, key := range []string{"TMPDIR", "TEMP", "TMP"} {
		if value, ok := os.LookupEnv(key); ok {
			keep = append(keep, key, value)
		}
	}
	NullEnvironment(t)
	kv = append(keep, kv...)
	for i := 0; i < len(kv); i += 2 {
		t.Setenv(kv[i], kv[i+1])
	}
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)
}
