package aircmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// aiTrainingWorkflowsPath is the AiTrainingService index of the caller's own AIR
// runs. It returns cheap (job_run_id, submit_time) pairs, letting `air list`
// order and page without scanning the Jobs runs/list firehose. This is called
// with a raw client.Do because the SDK does not model the AiTrainingService.
const aiTrainingWorkflowsPath = "/api/2.0/ai-training/workflows"

// workflowRef is one run from the index: its Jobs run id and submission time.
type workflowRef struct {
	jobRunID     int64
	submitTimeMs int64
}

type aiTrainingWorkflow struct {
	// job_run_id is a Jobs run id; tolerate it arriving as a JSON number or string.
	JobRunID json.Number `json:"job_run_id"`
	// submit_time is a proto Timestamp, serialized over HTTP as either an RFC3339
	// string or a {seconds, nanos} object.
	SubmitTime json.RawMessage `json:"submit_time"`
}

type aiTrainingWorkflowsResponse struct {
	TrainingWorkflows []aiTrainingWorkflow `json:"training_workflows"`
	NextPageToken     string               `json:"next_page_token"`
}

// listAiTrainingWorkflows pages the index and returns every workflow ref the
// caller owns. Pagination stops at the end or when a page token repeats, which
// guards against a stuck or cycling cursor without an arbitrary page cap.
func listAiTrainingWorkflows(ctx context.Context, w *databricks.WorkspaceClient, activeOnly bool) ([]workflowRef, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var refs []workflowRef
	seenTokens := map[string]bool{}
	// The index can return the same job_run_id on more than one page; dedupe so
	// the newest-`limit` truncation counts unique runs, not repeats.
	seenIDs := map[int64]bool{}
	var pageToken string
	for {
		query := map[string]any{}
		if activeOnly {
			query["active_only"] = true
		}
		if pageToken != "" {
			query["page_token"] = pageToken
		}

		var resp aiTrainingWorkflowsResponse
		err = apiClient.Do(ctx, http.MethodGet, aiTrainingWorkflowsPath, nil, nil, query, &resp)
		if err != nil {
			return nil, fmt.Errorf("failed to list training workflows: %w", err)
		}

		for _, wf := range resp.TrainingWorkflows {
			id, err := wf.JobRunID.Int64()
			if err != nil || id == 0 || seenIDs[id] {
				continue
			}
			seenIDs[id] = true
			refs = append(refs, workflowRef{jobRunID: id, submitTimeMs: parseSubmitTimeMs(wf.SubmitTime)})
		}

		if resp.NextPageToken == "" || seenTokens[resp.NextPageToken] {
			break
		}
		seenTokens[resp.NextPageToken] = true
		pageToken = resp.NextPageToken
	}
	return refs, nil
}

// parseSubmitTimeMs converts a proto Timestamp (RFC3339 string or {seconds, nanos}
// object) to epoch milliseconds, or 0 when absent or unparseable (so it sorts last).
func parseSubmitTimeMs(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}

	var s string
	if json.Unmarshal(raw, &s) == nil {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.UnixMilli()
		}
		return 0
	}

	var obj struct {
		Seconds int64 `json:"seconds"`
		Nanos   int64 `json:"nanos"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Seconds*1000 + obj.Nanos/1_000_000
	}
	return 0
}
