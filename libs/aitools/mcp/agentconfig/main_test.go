package agentconfig

import (
	"os"
	"os/exec"
	"testing"

	"github.com/databricks/cli/internal/testutil/hostenv"
	"github.com/databricks/cli/libs/aitools/agents"
)

func TestMain(m *testing.M) {
	// No agent CLI is on PATH unless a test puts one there, so no test runs a host CLI.
	agents.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	finish := hostenv.Isolate()
	os.Exit(finish(m.Run()))
}
