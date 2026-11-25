package aircmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// otherCommandConfig differs from minimalConfig in content, so its launch dir is
// not remembered and its files are uploaded.
var otherCommandConfig = strings.Replace(minimalConfig, "command: python train.py", "command: python other.py", 1)

func submitWithConfig(t *testing.T, w *databricks.WorkspaceClient, yaml, key string) error {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", yaml)
	cfg, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	_, _, err = submitWorkload(t.Context(), w, cfg, cfgPath, key, false)
	return err
}

// warmUserNameCache runs one launch with a caller key, which takes the normal path
// and caches the user name for the credential.
func warmUserNameCache(t *testing.T, w *databricks.WorkspaceClient) {
	t.Helper()
	require.NoError(t, submitMinimal(t, w, "idem-warm"))
}

func indexOfImport(events []string) int {
	for i, e := range events {
		if strings.HasPrefix(e, "import") {
			return i
		}
	}
	return -1
}

func lastIndexOfImport(events []string) int {
	last := -1
	for i, e := range events {
		if strings.HasPrefix(e, "import") {
			last = i
		}
	}
	return last
}

// On a cache hit with the flag on, the run is submitted before any file is uploaded,
// and no SCIM call precedes it.
func TestSpeculativeSubmitRunsBeforeUploadOnCacheHit(t *testing.T) {
	t.Setenv(speculativeSubmitEnv, "1")
	w, s := newRelaunchClient(t)
	warmUserNameCache(t, w)
	s.take()

	require.NoError(t, submitWithConfig(t, w, otherCommandConfig, ""))
	events := s.take()

	assert.NotContains(t, events, "me", "a cache hit must not call Me")
	submitAt := slices.Index(events, "submit")
	require.GreaterOrEqual(t, submitAt, 0, events)
	require.GreaterOrEqual(t, indexOfImport(events), 0, "the launch files must still be uploaded: %v", events)
	assert.Less(t, submitAt, indexOfImport(events), events)
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.submitPaths, 2)
	assert.True(t, strings.HasPrefix(s.submitPaths[1], "/Workspace/Users/a@example.com/.air/cli_launch/"), s.submitPaths[1])
	assert.Empty(t, s.cancelledRuns)
}

// Without a cached name, the launch keeps the normal order: Me, uploads, then submit.
func TestSpeculativeSubmitMissKeepsNormalOrder(t *testing.T) {
	t.Setenv(speculativeSubmitEnv, "1")
	w, s := newRelaunchClient(t)

	require.NoError(t, submitWithConfig(t, w, otherCommandConfig, ""))
	events := s.take()

	me := slices.Index(events, "me")
	require.GreaterOrEqual(t, me, 0, events)
	assert.Less(t, me, indexOfImport(events), events)
	assert.Greater(t, slices.Index(events, "submit"), lastIndexOfImport(events), events)
}

// With the flag off, a cache hit still uploads before submitting.
func TestSpeculativeSubmitOffByDefault(t *testing.T) {
	w, s := newRelaunchClient(t)
	warmUserNameCache(t, w)
	s.take()

	require.NoError(t, submitWithConfig(t, w, otherCommandConfig, ""))
	events := s.take()

	assert.NotContains(t, events, "me")
	assert.Greater(t, slices.Index(events, "submit"), lastIndexOfImport(events), events)
}

// A caller's idempotency key keeps the normal order even with the flag on, because
// a cancelled speculative run would leave that key spent.
func TestSpeculativeSubmitCallerKeyKeepsNormalOrder(t *testing.T) {
	t.Setenv(speculativeSubmitEnv, "1")
	w, s := newRelaunchClient(t)
	warmUserNameCache(t, w)
	s.take()

	require.NoError(t, submitWithConfig(t, w, otherCommandConfig, "caller-key"))
	events := s.take()

	assert.Greater(t, slices.Index(events, "submit"), lastIndexOfImport(events), events)
	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Equal(t, "caller-key", s.tokens[len(s.tokens)-1])
}

// A config error found while the speculative run is in flight cancels that run,
// and the caller sees the server's message.
func TestSpeculativeSubmitCancelsRunWhenValidationFails(t *testing.T) {
	t.Setenv(speculativeSubmitEnv, "1")
	w, s := newRelaunchClient(t)
	warmUserNameCache(t, w)
	s.take()
	s.setValidateFailure("bad command")

	err := submitWithConfig(t, w, otherCommandConfig, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad command")

	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Equal(t, []int64{777}, s.cancelledRuns)
}

// A failed upload cancels the speculative run, so no run is left without its files.
func TestSpeculativeSubmitCancelsRunWhenStagingFails(t *testing.T) {
	t.Setenv(speculativeSubmitEnv, "1")
	w, s := newRelaunchClient(t)
	warmUserNameCache(t, w)
	s.take()
	s.setFailImport(true)

	require.Error(t, submitWithConfig(t, w, otherCommandConfig, ""))

	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Equal(t, []int64{777}, s.cancelledRuns)
}
