package pipelines

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/testserver"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockPipelineID = "orders-pipeline-id"
	mockUpdateID   = "mock-update-id"
)

func capturedFixtureEvents(t *testing.T) []testPipelineEvent {
	t.Helper()

	raw, err := os.ReadFile("testdata/pipeline_test_events.json")
	require.NoError(t, err)
	var response struct {
		Events []testPipelineEvent `json:"events"`
	}
	require.NoError(t, json.Unmarshal(raw, &response))
	return response.Events
}

func TestPipelineTestCommandWithMockedControlPlane(t *testing.T) {
	// StartUpdate and GetUpdate use their normal HTTP contracts. Only the
	// DP-to-CP event synchronization result is substituted with events captured
	// from a real DP run.
	server := testserver.New(t)
	var startRequest startTestUpdateRequest
	fixture := capturedFixtureEvents(t)
	split := len(fixture) / 2
	eventRequests := 0

	server.Handle(http.MethodPost, "/api/2.0/pipelines/"+mockPipelineID+"/updates", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &startRequest))
		return pipelines.StartUpdateResponse{UpdateId: mockUpdateID}
	})
	server.Handle(http.MethodGet, "/api/2.0/pipelines/"+mockPipelineID+"/updates/"+mockUpdateID, func(_ testserver.Request) any {
		return pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				UpdateId: mockUpdateID,
				State:    pipelines.UpdateInfoStateCompleted,
			},
		}
	})
	server.Handle(http.MethodGet, "/api/2.0/pipelines/"+mockPipelineID+"/events", func(req testserver.Request) any {
		eventRequests++
		assert.Equal(t, "update_id = '"+mockUpdateID+"'", req.URL.Query().Get("filter"))
		assert.Equal(t, "timestamp asc", req.URL.Query().Get("order_by"))
		assert.Equal(t, "250", req.URL.Query().Get("max_results"))
		if req.URL.Query().Get("page_token") == "" {
			return map[string]any{
				"events":          fixture[:split],
				"next_page_token": "page-2",
			}
		}
		assert.Equal(t, "page-2", req.URL.Query().Get("page_token"))
		return map[string]any{"events": fixture[split:]}
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", testserver.UserNameTokenPrefix+strings.Repeat("a", 32))
	ctx := root.SkipLoadBundle(t.Context())

	var stdout, stderr bytes.Buffer
	cli := root.New(ctx)
	pipelinesCommand := &cobra.Command{Use: "pipelines"}
	pipelinesCommand.AddCommand(testCommand())
	cli.AddCommand(pipelinesCommand)
	cli.SetOut(&stdout)
	cli.SetErr(&stderr)
	cli.SetArgs([]string{
		"pipelines", "test",
		"--pipeline-id", mockPipelineID,
	})
	err := root.Execute(ctx, cli)

	require.Error(t, err)
	assert.Equal(t, startTestUpdateRequest{TestOnly: true}, startRequest)
	assert.Equal(t, 2, eventRequests)
	assert.Contains(t, err.Error(), "2 of 4 pipeline tests failed")
	assert.Contains(t, stdout.String(), "Pipeline test progress for "+mockPipelineID+" (update "+mockUpdateID+")")
	assert.Contains(t, stdout.String(), "RUN    test_demo.py::test_pass")
	assert.Contains(t, stdout.String(), "PASS   test_demo.py::test_pass")
	assert.Contains(t, stdout.String(), "FAIL   test_demo.py::test_fail")
	assert.Contains(t, stdout.String(), "AssertionError: intentional demo failure")
	assert.True(t, strings.HasSuffix(stdout.String(), "4 tests: 1 passed, 1 failed, 1 skipped, 1 errors\n"))

	t.Logf("mocked CLI demo:\n$ databricks pipelines test --pipeline-id %s\n%s", mockPipelineID, stdout.String())
}

func TestPollWaitsForDelayedDPToCPEventSync(t *testing.T) {
	server := testserver.New(t)
	fixture := capturedFixtureEvents(t)
	eventRequests := 0

	server.Handle(http.MethodGet, "/api/2.0/pipelines/"+mockPipelineID+"/updates/"+mockUpdateID, func(_ testserver.Request) any {
		return pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				UpdateId: mockUpdateID,
				State:    pipelines.UpdateInfoStateCompleted,
			},
		}
	})
	server.Handle(http.MethodGet, "/api/2.0/pipelines/"+mockPipelineID+"/events", func(_ testserver.Request) any {
		eventRequests++
		if eventRequests == 1 {
			// The update is terminal, but the summary has not crossed the
			// DP-to-CP sync boundary yet.
			return map[string]any{"events": fixture[:len(fixture)-1]}
		}
		return map[string]any{"events": fixture}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: testserver.UserNameTokenPrefix + strings.Repeat("b", 32),
	})
	require.NoError(t, err)

	var streamedSnapshots int
	_, events, err := pollTestEventsUntilTerminal(
		t.Context(),
		w,
		mockPipelineID,
		mockUpdateID,
		time.Millisecond,
		time.Millisecond,
		100*time.Millisecond,
		func([]testPipelineEvent) error {
			streamedSnapshots++
			return nil
		},
	)
	require.NoError(t, err)
	require.NoError(t, reduceTestEvents(events).checkComplete())
	assert.Equal(t, 2, eventRequests)
	assert.Equal(t, 2, streamedSnapshots)
}

func TestPipelineTestNoWaitJSON(t *testing.T) {
	server := testserver.New(t)
	server.Handle(http.MethodPost, "/api/2.0/pipelines/"+mockPipelineID+"/updates", func(_ testserver.Request) any {
		return pipelines.StartUpdateResponse{UpdateId: mockUpdateID}
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", testserver.UserNameTokenPrefix+strings.Repeat("c", 32))
	ctx := root.SkipLoadBundle(t.Context())

	var stdout, stderr bytes.Buffer
	cli := root.New(ctx)
	pipelinesCommand := &cobra.Command{Use: "pipelines"}
	pipelinesCommand.AddCommand(testCommand())
	cli.AddCommand(pipelinesCommand)
	cli.SetOut(&stdout)
	cli.SetErr(&stderr)
	cli.SetArgs([]string{
		"pipelines", "test",
		"--pipeline-id", mockPipelineID,
		"--no-wait",
		"-o", "json",
	})

	require.NoError(t, root.Execute(ctx, cli))
	var got testSubmissionJSONOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	assert.Equal(t, testSubmissionJSONOutput{
		PipelineID: mockPipelineID,
		UpdateID:   mockUpdateID,
		Submitted:  true,
	}, got)
	assert.Empty(t, stderr.String())
}
