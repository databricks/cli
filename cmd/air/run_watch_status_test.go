package aircmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// watchJSONEvents summarizes `air run --watch -o json` output: the ordered
// SUBMITTED/STATUS events (STATUS as "current<-previous"), the LOG count, and
// the closing envelope's status.
type watchJSONEvents struct {
	lifecycle   []string
	logCount    int
	finalStatus string
}

func parseWatchJSONEvents(t *testing.T, out string) watchJSONEvents {
	t.Helper()
	var got watchJSONEvents
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		var ev struct {
			Type     string `json:"type"`
			Status   string `json:"status"`
			Previous string `json:"previous_status"`
			Data     struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &ev), "JSONL line must parse: %s", line)
		switch ev.Type {
		case "SUBMITTED":
			got.lifecycle = append(got.lifecycle, "SUBMITTED")
		case "STATUS":
			got.lifecycle = append(got.lifecycle, ev.Status+"<-"+ev.Previous)
		case "LOG":
			got.logCount++
		}
		if ev.Data.Status != "" {
			got.finalStatus = ev.Data.Status
		}
	}
	return got
}

// A run cancelled while PENDING ends the Bricklens stream empty and hands off
// to MLflow, which follows from the pre-stream PENDING status. The consumer must
// still see each transition once and never PENDING after CANCELED, while the
// MLflow grace re-read keeps printing artifacts that appear only after the
// cancel, uncapped.
func TestRunWatchJSONStatusAfterEmptyBricklensHandoff(t *testing.T) {
	oldBricklens, oldRetry := bricklensPollInterval, retryCheckInterval
	bricklensPollInterval, retryCheckInterval = 10*time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { bricklensPollInterval, retryCheckInterval = oldBricklens, oldRetry })

	var canceled atomic.Bool
	var outputCalls atomic.Int32
	server := testserver.New(t)
	defer server.Close()
	server.Handle("POST", "/api/2.0/ai-training/config:validate", func(testserver.Request) any { return `{}` })
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(testserver.Request) any { return `{"run_id":777}` })
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(testserver.Request) any {
		if canceled.Load() {
			return `{"run_id":777,"state":{"life_cycle_state":"TERMINATED","result_state":"CANCELED","state_message":"Run cancelled by user"},"tasks":[{"run_id":778}]}`
		}
		return `{"run_id":777,"state":{"life_cycle_state":"PENDING"},"tasks":[{"run_id":778}]}`
	})
	server.Handle("GET", "/api/2.0/ai-training/workflows/by-run-id/777/logs", func(testserver.Request) any {
		canceled.Store(true)
		return `{"log_records":[]}`
	})
	// The MLflow IDs only appear on the second lookup, after the grace wait.
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(testserver.Request) any {
		if outputCalls.Add(1) == 1 {
			return `{}`
		}
		return `{"ai_runtime_task_output":{"mlflow_experiment_id":"exp1","mlflow_run_id":"run1"}}`
	})
	server.Handle("GET", "/api/2.0/mlflow/runs/get", func(testserver.Request) any { return `{}` })
	server.Handle("GET", "/api/2.0/mlflow/artifacts/list", func(req testserver.Request) any {
		if req.URL.Query().Get("path") == "logs" {
			return `{"files":[{"path":"logs/node_0","is_dir":true}]}`
		}
		return `{"files":[{"path":"logs/node_0/logs-0.chunk.txt"},{"path":"logs/node_0/logs-1.chunk.txt"}]}`
	})
	server.Handle("GET", "/api/2.0/mlflow/artifacts/credentials-for-read", func(req testserver.Request) any {
		chunk := "zero"
		if strings.HasSuffix(req.URL.Query().Get("path"), "logs-1.chunk.txt") {
			chunk = "one"
		}
		return fmt.Sprintf(`{"credential_infos":[{"signed_uri":%q}]}`, server.URL+"/artifact/"+chunk)
	})
	server.Handle("GET", "/artifact/zero", func(testserver.Request) any { return strings.Repeat("chunk-zero\n", 10001) })
	server.Handle("GET", "/artifact/one", func(testserver.Request) any { return "chunk-one\n" })
	testserver.AddDefaultHandlers(server)

	var out bytes.Buffer
	err := runWatchCmd(t, flags.OutputJSON, &out, server.URL)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	got := parseWatchJSONEvents(t, out.String())
	assert.Equal(t, []string{"SUBMITTED", "PENDING<-", "CANCELED<-PENDING"}, got.lifecycle)
	assert.GreaterOrEqual(t, outputCalls.Load(), int32(2), "the MLflow handoff must re-read after its grace wait")
	assert.Equal(t, 10002, got.logCount, "late chunks must print in full, not as a capped tail")
	assert.Contains(t, out.String(), `"line":"chunk-one"`)
	assert.Equal(t, "CANCELED", got.finalStatus)
}

// FEATURE_DISABLED hands off to MLflow on the first poll, again following from
// the pre-stream status; PENDING must not be reported twice.
func TestRunWatchJSONStatusAfterFeatureDisabledHandoff(t *testing.T) {
	oldRetry := retryCheckInterval
	retryCheckInterval = 10 * time.Millisecond
	t.Cleanup(func() { retryCheckInterval = oldRetry })

	var canceled atomic.Bool
	server := testserver.New(t)
	defer server.Close()
	server.Handle("POST", "/api/2.0/ai-training/config:validate", func(testserver.Request) any { return `{}` })
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(testserver.Request) any { return `{"run_id":777}` })
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(testserver.Request) any {
		if canceled.Load() {
			return `{"run_id":777,"state":{"life_cycle_state":"TERMINATED","result_state":"CANCELED"},"tasks":[{"run_id":778}]}`
		}
		return `{"run_id":777,"state":{"life_cycle_state":"PENDING"},"tasks":[{"run_id":778}]}`
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(testserver.Request) any { return `{}` })
	server.Handle("GET", "/api/2.0/ai-training/workflows/by-run-id/777/logs", func(testserver.Request) any {
		canceled.Store(true)
		return testserver.Response{StatusCode: 403, Body: `{"error_code":"FEATURE_DISABLED","message":"disabled"}`}
	})
	testserver.AddDefaultHandlers(server)

	var out bytes.Buffer
	err := runWatchCmd(t, flags.OutputJSON, &out, server.URL)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	got := parseWatchJSONEvents(t, out.String())
	assert.Equal(t, []string{"SUBMITTED", "PENDING<-", "CANCELED<-PENDING"}, got.lifecycle)
	assert.Equal(t, "CANCELED", got.finalStatus)
}

// An active run whose logs turn up in MLflow hands off while RUNNING. The
// handoff must report what Bricklens last saw, not regress to the PENDING
// status resolved before streaming.
func TestRunWatchJSONStatusAfterActiveMLflowProbeHandoff(t *testing.T) {
	oldBricklens, oldRetry := bricklensPollInterval, retryCheckInterval
	bricklensPollInterval, retryCheckInterval = time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { bricklensPollInterval, retryCheckInterval = oldBricklens, oldRetry })

	var statusCalls atomic.Int32
	var finished atomic.Bool
	server := testserver.New(t)
	defer server.Close()
	server.Handle("POST", "/api/2.0/ai-training/config:validate", func(testserver.Request) any { return `{}` })
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(testserver.Request) any { return `{"run_id":777}` })
	server.Handle("GET", "/api/2.2/jobs/runs/get", func(testserver.Request) any {
		switch {
		case finished.Load():
			return `{"run_id":777,"state":{"life_cycle_state":"TERMINATED","result_state":"SUCCESS"},"tasks":[{"run_id":778}]}`
		case statusCalls.Add(1) == 1:
			return `{"run_id":777,"state":{"life_cycle_state":"PENDING"},"tasks":[{"run_id":778}]}`
		default:
			return `{"run_id":777,"state":{"life_cycle_state":"RUNNING"},"tasks":[{"run_id":778}]}`
		}
	})
	server.Handle("GET", "/api/2.0/ai-training/workflows/by-run-id/777/logs", func(testserver.Request) any { return `{"log_records":[]}` })
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(testserver.Request) any {
		return `{"ai_runtime_task_output":{"mlflow_experiment_id":"exp1","mlflow_run_id":"run1"}}`
	})
	server.Handle("GET", "/api/2.0/mlflow/runs/get", func(testserver.Request) any { return `{}` })
	server.Handle("GET", "/api/2.0/mlflow/artifacts/list", func(req testserver.Request) any {
		if req.URL.Query().Get("path") == "logs" {
			return `{"files":[{"path":"logs/node_0","is_dir":true}]}`
		}
		return `{"files":[{"path":"logs/node_0/logs-0.chunk.txt"}]}`
	})
	server.Handle("GET", "/api/2.0/mlflow/artifacts/credentials-for-read", func(testserver.Request) any {
		return fmt.Sprintf(`{"credential_infos":[{"signed_uri":%q}]}`, server.URL+"/artifact/zero")
	})
	server.Handle("GET", "/artifact/zero", func(testserver.Request) any {
		finished.Store(true)
		return "hello\n"
	})
	testserver.AddDefaultHandlers(server)

	var out bytes.Buffer
	require.NoError(t, runWatchCmd(t, flags.OutputJSON, &out, server.URL))

	got := parseWatchJSONEvents(t, out.String())
	assert.Equal(t, []string{"SUBMITTED", "PENDING<-", "RUNNING<-PENDING", "SUCCESS<-RUNNING"}, got.lifecycle)
	assert.Equal(t, 1, got.logCount)
	assert.Equal(t, "SUCCESS", got.finalStatus)
}

func TestStatusTransitionReporterDropsRepeatsAndPostTerminalStates(t *testing.T) {
	var got []string
	report := newStatusTransitionReporter(func(current, previous string) {
		got = append(got, current+"<-"+previous)
	})

	// Each caller passes its own local previous state; the reporter substitutes
	// the last state it actually reported.
	report("PENDING", "")
	report("PENDING", "")
	report("RUNNING", "PENDING")
	report("CANCELED", "")
	report("PENDING", "")
	report("CANCELED", "PENDING")

	assert.Equal(t, []string{"PENDING<-", "RUNNING<-PENDING", "CANCELED<-RUNNING"}, got)
}
