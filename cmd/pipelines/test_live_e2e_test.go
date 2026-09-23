package pipelines

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

// TestLiveE2EAgainstCP exercises the real command read+fold+render path against
// a live control plane. It is guarded by SDP_LIVE_E2E=1 (skipped otherwise) so
// it never runs in CI. Env inputs: DATABRICKS_HOST, DATABRICKS_TOKEN,
// SDP_PIPELINE_ID, SDP_UPDATE_ID.
//
// It reads the update's /events over the external REST surface exactly as the
// `databricks pipelines test` command does (fetchTestEvents -> filterTestEvents
// -> reduceTestEvents -> render*), and prints what the CLI would emit.
func TestLiveE2EAgainstCP(t *testing.T) {
	if os.Getenv("SDP_LIVE_E2E") != "1" {
		t.Skip("set SDP_LIVE_E2E=1 (+ DATABRICKS_HOST/TOKEN, SDP_PIPELINE_ID, SDP_UPDATE_ID) to run")
	}
	pid := os.Getenv("SDP_PIPELINE_ID")
	updateID := os.Getenv("SDP_UPDATE_ID")
	require.NotEmpty(t, pid)
	require.NotEmpty(t, updateID)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  os.Getenv("DATABRICKS_HOST"),
		Token: os.Getenv("DATABRICKS_TOKEN"),
	})
	require.NoError(t, err)

	ctx := t.Context()
	events, err := fetchTestEvents(ctx, w, pid, updateID)
	require.NoError(t, err)

	fmt.Printf("LIVE-E2E: external /events returned %d test-typed events for update %s\n", len(events), updateID)
	result := reduceTestEvents(events)
	require.NoError(t, result.checkComplete())
	var human bytes.Buffer
	require.NoError(t, renderTestResultsText(&human, pid, updateID, result))
	fmt.Print("LIVE-E2E human:\n", human.String())

	var jsonBuf bytes.Buffer
	enc := json.NewEncoder(&jsonBuf)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(result.toJSONOutput(pid, updateID)))
	fmt.Print("LIVE-E2E json:\n", jsonBuf.String())
}
