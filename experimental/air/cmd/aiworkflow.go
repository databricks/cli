package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
)

// listWorkflowsPath is the ListTrainingWorkflows binding. It is
// PUBLIC_UNDOCUMENTED, so we call it directly like the endpoints in mlflow.go.
const listWorkflowsPath = "/api/2.0/ai-training/workflows"

// trainingWorkflowsResponse is the ListTrainingWorkflows response. Only the
// fields the list table and JSON envelope consume are modeled.
type trainingWorkflowsResponse struct {
	TrainingWorkflows []trainingWorkflow `json:"training_workflows"`
	NextPageToken     string             `json:"next_page_token"`
}

type trainingWorkflow struct {
	JobRunID string `json:"job_run_id"`

	TaskRunID string `json:"task_run_id"`
	Metadata  struct {
		CreatorName string `json:"creator_name"`
	} `json:"metadata"`
	Spec struct {
		Compute struct {
			HardwareAcceleratorType string `json:"hardware_accelerator_type"`
			AcceleratorCount        int    `json:"accelerator_count"`
		} `json:"compute"`
	} `json:"spec"`
	Status struct {
		State     string `json:"state"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Job       struct {
			Name string `json:"name"`
		} `json:"job"`
		Mlflow struct {
			Experiment   string `json:"experiment"`
			ExperimentID string `json:"experiment_id"`
			RunID        string `json:"run_id"`
		} `json:"mlflow"`
	} `json:"status"`
}

// listTrainingWorkflows calls the ListTrainingWorkflows RPC with the given query
// parameters.
func listTrainingWorkflows(ctx context.Context, w *databricks.WorkspaceClient, query map[string]any) (*trainingWorkflowsResponse, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	var resp trainingWorkflowsResponse
	err = apiClient.Do(ctx, http.MethodGet, listWorkflowsPath, nil, nil, query, &resp)
	if err != nil {
		// A server error can arrive with an empty body, leaving %w blank, so
		// surface the HTTP status to make the failure diagnosable.
		if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
			switch apiErr.StatusCode {
			case http.StatusGatewayTimeout, http.StatusServiceUnavailable, http.StatusRequestTimeout:
				// Trailing %w keeps the error chain; the body is usually empty.
				return nil, fmt.Errorf("timed out listing runs (HTTP %d): --all-users makes the server list every user's runs, which can exceed the gateway timeout on a busy workspace. Add --active to list only active runs, or drop --all-users to list your own.%w", apiErr.StatusCode, err)
			}
			return nil, fmt.Errorf("failed to list training workflows (HTTP %d): %w", apiErr.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to list training workflows: %w", err)
	}
	return &resp, nil
}
