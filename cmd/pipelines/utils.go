package pipelines

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	configresources "github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// Copied from cmd/bundle/run.go
// promptRunnablePipeline prompts the user to select a runnable pipeline.
func promptRunnablePipeline(ctx context.Context, b *bundle.Bundle) (string, error) {
	// Compute map of "Human readable name of resource" -> "resource key".
	inv := make(map[string]string)
	for k, ref := range resources.Completions(b, run.IsRunnable) {
		title := fmt.Sprintf("%s: %s", ref.Description.SingularTitle, ref.Resource.GetName())
		inv[title] = k
	}

	key, err := cmdio.Select(ctx, inv, "Select a pipeline")
	if err != nil {
		return "", err
	}

	return key, nil
}

// autoSelectSinglePipeline checks if there's exactly one pipeline resource in the bundle and returns its key.
// Returns empty string if there's not exactly one pipeline.
func autoSelectSinglePipeline(b *bundle.Bundle) string {
	completions := resources.Completions(b, run.IsRunnable)
	if len(completions) == 1 {
		for key, ref := range completions {
			if _, ok := ref.Resource.(*configresources.Pipeline); ok {
				return key
			}
		}
	}
	return ""
}

// Copied from cmd/bundle/run.go
// resolveRunArgument resolves the resource key to run
// Returns the remaining arguments to pass to the runner, if applicable.
// When no arguments are specified, auto-selects a pipeline if there's exactly one,
// otherwise prompts the user to select a pipeline.
func resolveRunArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, []string, error) {
	if len(args) == 0 {
		if key := autoSelectSinglePipeline(b); key != "" {
			return key, args, nil
		}

		if cmdio.IsPromptSupported(ctx) {
			key, err := promptRunnablePipeline(ctx, b)
			if err != nil {
				return "", nil, err
			}
			return key, args, nil
		}
	}

	if len(args) < 1 {
		return "", nil, errors.New("expected a KEY of the resource to run")
	}

	return args[0], args[1:], nil
}

// Copied from cmd/bundle/run.go
// keyToRunner converts a resource key to a runner.
func keyToRunner(b *bundle.Bundle, arg string) (run.Runner, error) {
	// Locate the resource to run.
	ref, err := resources.Lookup(b, arg, run.IsRunnable)
	if err != nil {
		return nil, err
	}

	// Convert the resource to a runnable resource.
	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

type PipelineEventsResponse struct {
	Events        []pipelines.PipelineEvent `json:"events"`
	NextPageToken string                    `json:"next_page_token,omitempty"`
}

type PipelineEventsQueryParams struct {
	Filter     string `json:"filter,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
	PageToken  string `json:"page_token,omitempty"`
	OrderBy    string `json:"order_by,omitempty"`
}

func fetchAllPipelineEvents(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string, params *PipelineEventsQueryParams) ([]pipelines.PipelineEvent, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/api/2.0/pipelines/%s/events", pipelineID)

	queryParams := map[string]string{}
	if params.Filter != "" {
		queryParams["filter"] = params.Filter
	}
	if params.MaxResults > 0 {
		queryParams["max_results"] = fmt.Sprintf("%d", params.MaxResults)
	}

	if params.OrderBy != "" {
		queryParams["order_by"] = params.OrderBy
	}

	var response PipelineEventsResponse
	err = apiClient.Do(
		ctx,
		"GET",
		path,
		nil,
		nil,
		queryParams,
		&response,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline events: %w", err)
	}

	return response.Events, nil
}
