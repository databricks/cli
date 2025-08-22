package pipelines

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	configresources "github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// Copied from cmd/bundle/run.go
// promptResource prompts the user to select a pipeline.
// If filter is provided, only resources that pass the filter will be included.
func promptResource(ctx context.Context, b *bundle.Bundle, filters ...resources.Filter) (string, error) {
	// Compute map of "Human readable name of resource" -> "resource key".
	inv := make(map[string]string)
	completions := resources.Completions(b, filters...)

	for k, ref := range completions {
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
			key, err := promptResource(ctx, b, run.IsRunnable)
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

// resolvePipelineArgument auto-selects a pipeline if there's exactly one and no arguments are specified,
// otherwise prompts the user to select a pipeline.
func resolvePipelineArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	if key := autoSelectSinglePipeline(b); key != "" {
		return key, nil
	}

	if cmdio.IsPromptSupported(ctx) {
		return promptResource(ctx, b, run.IsRunnable, func(ref resources.Reference) bool {
			_, ok := ref.Resource.(*configresources.Pipeline)
			return ok
		})
	}
	return "", errors.New("expected a KEY of the pipeline")
}

// resolvePipelineIdFromKey resolves a pipeline KEY to its actual pipeline ID.
// This function handles the bundle initialization, resource lookup, and pipeline ID extraction.
func resolvePipelineIdFromKey(ctx context.Context, b *bundle.Bundle, key string) (string, error) {
	ref, err := resources.Lookup(b, key)
	if err != nil {
		return "", err
	}

	pipeline, ok := ref.Resource.(*configresources.Pipeline)
	if !ok {
		return "", fmt.Errorf("resource %s is not a pipeline", key)
	}

	pipelineId := pipeline.ID
	if pipelineId == "" {
		return "", fmt.Errorf("pipeline ID for pipeline %s is not found", ref.Key)
	}

	return pipelineId, nil
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

// formatOSSTemplateWarningMessage formats the warning message for OSS template pipeline YAML files.
func formatOSSTemplateWarningMessage(d diag.Diagnostic) string {
	fileName := "A pipeline YAML file"
	if len(d.Locations) > 0 && d.Locations[0].File != "" {
		fileName = d.Locations[0].File
	}

	return fileName + ` seems to be formatted for open-source Spark Declarative Pipelines.
Pipelines CLI currently only supports Lakeflow Declarative Pipelines development.
To see an example of a supported pipelines template, create a new Pipelines CLI project with "pipelines init".`
}

// checkForOSSTemplateWarning checks for a warning in the logs that suggests a pipelines YAML file
// is formatted for open-source Spark Declarative Pipelines and returns an error if found.
// For the Spark Declarative Pipelines template, see: https://github.com/apache/spark/blob/master/python/pyspark/pipelines/init_cli.py
// Logs all collected diagnostics to expose them.
func checkForOSSTemplateWarning(ctx context.Context, diags diag.Diagnostics) error {
	var ossWarning *diag.Diagnostic

	for _, d := range diags {
		// The "definitions" field in the root is expected in OSS Spark Declarative Pipelines templates but is not supported in the Pipelines CLI.
		if d.Severity == diag.Warning && strings.Contains(d.Summary, "unknown field: definitions") && len(d.Paths) == 1 && d.Paths[0].Equal(dyn.EmptyPath) {
			ossWarning = &d
		}
		logdiag.LogDiag(ctx, d)
	}

	if ossWarning != nil {
		return errors.New(formatOSSTemplateWarningMessage(*ossWarning))
	}

	return nil
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

// fetchAllPipelineEvents retrieves pipeline events with optional SQL filtering and ordering.
// Necessary as current Go SDK endpoints don't support OrderBy parameter.
// Retrieves only one page of results, so the number of results is bound by the API's limit of results per page.
func fetchAllPipelineEvents(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string, params *PipelineEventsQueryParams) ([]pipelines.PipelineEvent, error) {
	maxResultsPerPage := 250
	if params.MaxResults > maxResultsPerPage {
		return nil, fmt.Errorf("number of results must be %d or less", maxResultsPerPage)
	}

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
		queryParams["max_results"] = strconv.Itoa(params.MaxResults)
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

// getMostRecentUpdateId fetches one page of updates for a given pipeline and returns the first update ID.
// Expects to receive updates in decreasing timestamp order, so the first update is the most recent.
func getMostRecentUpdateId(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string) (string, error) {
	request := pipelines.ListUpdatesRequest{
		PipelineId: pipelineID,
	}

	response, err := w.Pipelines.ListUpdates(ctx, request)
	if err != nil {
		return "", err
	}

	updates := response.Updates
	if len(updates) == 0 {
		return "", errors.New("no updates")
	}

	return updates[0].UpdateId, nil
}

// parseAndFormatTimestamp parses a timestamp string and formats it to the pipeline events API format.
func parseAndFormatTimestamp(timestamp string) (string, error) {
	if timestamp == "" {
		return "", nil
	}

	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return "", err
	}

	return t.Format("2006-01-02T15:04:05.000Z"), nil
}

// parseTimeToUnixMillis parses a time string and returns the number of milliseconds since epoch in UTC.
func parseTimeToUnixMillis(timeStr string) (*int64, error) {
	if timeStr == "" {
		return nil, nil
	}

	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid time format. Expected format: 2025-01-15T10:30:00Z (YYYY-MM-DDTHH:MM:SSZ), got: %s", timeStr)
	}
	unixMillis := t.UnixMilli()
	return &unixMillis, nil
}

// updatesBefore returns all updates with CreationTime <= ts
// Assumes updates are sorted in descending order, largest CreationTime first.
func updatesBefore(updates []pipelines.UpdateInfo, ts int64) []pipelines.UpdateInfo {
	// Binary search for index with CreationTime <= ts
	idx, _ := slices.BinarySearchFunc(updates, ts, func(u pipelines.UpdateInfo, target int64) int {
		if u.CreationTime <= target {
			return 1
		}
		return -1
	})

	return updates[idx:]
}

// updatesAfter returns all updates with CreationTime >= ts
// Assumes updates are sorted in descending order, largest CreationTime first.
func updatesAfter(updates []pipelines.UpdateInfo, ts int64) []pipelines.UpdateInfo {
	// Binary search for index with CreationTime < ts
	idx, _ := slices.BinarySearchFunc(updates, ts, func(u pipelines.UpdateInfo, target int64) int {
		if u.CreationTime < target {
			return 1
		}
		return -1
	})

	return updates[:idx]
}

// filterUpdates filters for updates within the startTime and endTime,
// assuming updates are in descending order, largest CreationTime first.
// Time is in milliseconds since epoch. If time is nil, it is ignored.
func filterUpdates(updates []pipelines.UpdateInfo, startTime, endTime *int64) ([]pipelines.UpdateInfo, error) {
	if (endTime == nil || updates[0].CreationTime <= *endTime) && (startTime == nil || updates[len(updates)-1].CreationTime >= *startTime) {
		return updates, nil
	}

	if startTime != nil && updates[0].CreationTime < *startTime {
		return nil, nil
	}

	if endTime != nil && updates[len(updates)-1].CreationTime > *endTime {
		return nil, nil
	}

	if startTime != nil {
		updates = updatesAfter(updates, *startTime)
	}

	if endTime != nil {
		updates = updatesBefore(updates, *endTime)
	}

	return updates, nil
}

// fetchPipelineUpdates fetches pipeline updates with optional filtering by time.
// Time is in milliseconds since epoch. If time is nil, it is ignored.
func fetchPipelineUpdates(ctx context.Context, w *databricks.WorkspaceClient, startTime, endTime *int64, pipelineId string) ([]pipelines.UpdateInfo, error) {
	var updates []pipelines.UpdateInfo
	var pageToken string

	for {
		request := pipelines.ListUpdatesRequest{
			PipelineId: pipelineId,
		}

		if pageToken != "" {
			request.PageToken = pageToken
		}

		response, err := w.Pipelines.ListUpdates(ctx, request)
		if err != nil {
			return nil, err
		}

		filteredUpdates, err := filterUpdates(response.Updates, startTime, endTime)
		if err != nil {
			return nil, err
		}
		updates = append(updates, filteredUpdates...)

		if response.NextPageToken == "" {
			break
		}
		pageToken = response.NextPageToken
	}

	return updates, nil
}
