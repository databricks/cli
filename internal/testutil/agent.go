package testutil

import "github.com/databricks/databricks-sdk-go/useragent"

// SetAgent makes useragent.AgentProvider report agent, or no agent if agent
// is empty. It clears the environment and the SDK's cached agent, so the agent
// running the tests does not leak into them.
// Note: use of this function is incompatible with parallel execution.
func SetAgent(t TestingT, agent string) {
	NullEnvironment(t)
	if agent != "" {
		t.Setenv("AGENT", agent)
	}
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)
}
