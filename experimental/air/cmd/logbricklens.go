package aircmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/databricks/databricks-sdk-go/client"
)

// bricklensLogsPathFmt is the log endpoint, keyed by Jobs run id. Called with a
// raw client.Do because the SDK does not model the AiTrainingService.
const bricklensLogsPathFmt = "/api/2.0/ai-training/workflows/by-run-id/%d/logs"

// logRecord is one log line from Bricklens.
type logRecord struct {
	// TimeUnixNano may arrive as a JSON number or string.
	TimeUnixNano json.Number `json:"time_unix_nano"`
	Body         string      `json:"body"`
	NodeIndex    int         `json:"node_index"`
}

// nano returns time_unix_nano as an int64, or 0 when absent or unparseable.
func (r logRecord) nano() int64 {
	n, err := r.TimeUnixNano.Int64()
	if err != nil {
		return 0
	}
	return n
}

type bricklensLogsResponse struct {
	LogRecords    []logRecord `json:"log_records"`
	NextPageToken string      `json:"next_page_token"`
}

// bricklensLogsQuery is the request-field surface of the log endpoint.
type bricklensLogsQuery struct {
	// fromSeconds and toSeconds bound the query window in Unix epoch seconds.
	fromSeconds int64
	toSeconds   int64
	pageToken   string
	pageSize    int
	// attemptNumber selects a retry attempt (0-indexed); -1 means latest.
	attemptNumber int
	nodeIndex     int
	// ascending returns oldest-first. The endpoint defaults to ascending when
	// absent, so the tail fetch must send an explicit false for newest-first.
	ascending bool
}

// getBricklensLogs fetches one page of logs. The API client is built once by the
// caller and reused across the poll loop. It returns the raw error so the caller
// can classify it via classifyLogError.
func getBricklensLogs(ctx context.Context, apiClient *client.DatabricksClient, runID int64, q bricklensLogsQuery) (*bricklensLogsResponse, error) {
	query := map[string]any{
		// Always sent: the tail path relies on an explicit false for newest-first.
		"ascending": strconv.FormatBool(q.ascending),
	}
	if q.fromSeconds > 0 {
		query["from"] = q.fromSeconds
	}
	if q.toSeconds > 0 {
		query["to"] = q.toSeconds
	}
	if q.pageToken != "" {
		query["page_token"] = q.pageToken
	}
	if q.pageSize > 0 {
		query["page_size"] = q.pageSize
	}
	if q.attemptNumber >= 0 {
		query["ref.attempt_number"] = q.attemptNumber
	}
	if q.nodeIndex >= 0 {
		query["filter.node_index"] = q.nodeIndex
	}

	var resp bricklensLogsResponse
	path := fmt.Sprintf(bricklensLogsPathFmt, runID)
	if err := apiClient.Do(ctx, http.MethodGet, path, nil, nil, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
