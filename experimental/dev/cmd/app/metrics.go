package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// AppMetrics holds aggregated metrics for an app.
type AppMetrics struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	ComputeSize  string            `json:"compute_size,omitempty"`
	URL          string            `json:"url,omitempty"`
	Deployments  DeploymentMetrics `json:"deployments"`
	LastUpdated  string            `json:"last_updated,omitempty"`
	StatusDetail string            `json:"status_detail,omitempty"`
}

// DeploymentMetrics holds deployment statistics.
type DeploymentMetrics struct {
	Total        int     `json:"total"`
	Succeeded    int     `json:"succeeded"`
	Failed       int     `json:"failed"`
	SuccessRate  float64 `json:"success_rate"`
	LastDeployed string  `json:"last_deployed,omitempty"`
}

func newMetricsCmd() *cobra.Command {
	var name string
	var timeRange string

	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Show metrics for an AppKit application",
		Long: `Show metrics and statistics for an AppKit application.

Displays information about app status, compute resources, and deployment history.

Examples:
  # Interactive mode - select app from picker
  databricks experimental appkit metrics

  # Show metrics for a specific app
  databricks experimental appkit metrics --name my-app

  # Show metrics with JSON output
  databricks experimental appkit metrics --name my-app --output json`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Prompt for app name if not provided
			if name == "" {
				selected, err := PromptForAppSelection(ctx, "Select an app to view metrics")
				if err != nil {
					return err
				}
				name = selected
			}

			w := cmdctx.WorkspaceClient(ctx)

			// Get app details
			app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: name})
			if err != nil {
				return fmt.Errorf("failed to get app: %w", err)
			}

			// Get deployment history
			deployments := w.Apps.ListDeployments(ctx, apps.ListAppDeploymentsRequest{
				AppName:  name,
				PageSize: 100,
			})

			var allDeployments []apps.AppDeployment
			for {
				d, err := deployments.Next(ctx)
				if err != nil {
					break
				}
				allDeployments = append(allDeployments, d)
			}

			// Calculate deployment metrics
			deploymentMetrics := calculateDeploymentMetrics(allDeployments, timeRange)

			// Build metrics response
			metrics := AppMetrics{
				Name:        app.Name,
				URL:         app.Url,
				Deployments: deploymentMetrics,
			}

			if app.ComputeStatus != nil {
				metrics.Status = string(app.ComputeStatus.State)
				metrics.StatusDetail = app.ComputeStatus.Message
			}

			if app.ComputeSize != "" {
				metrics.ComputeSize = string(app.ComputeSize)
			}

			return cmdio.Render(ctx, metrics)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the app to view metrics (prompts if not provided)")
	cmd.Flags().StringVar(&timeRange, "time-range", "7d", "Time range for metrics (1h, 24h, 7d, 30d)")

	return cmd
}

func calculateDeploymentMetrics(deployments []apps.AppDeployment, timeRange string) DeploymentMetrics {
	metrics := DeploymentMetrics{}

	if len(deployments) == 0 {
		return metrics
	}

	// Parse time range
	duration := parseTimeRange(timeRange)
	cutoff := time.Now().Add(-duration)

	var succeeded, failed int
	var lastDeployed time.Time

	for _, d := range deployments {
		if d.Status == nil {
			continue
		}

		// Check if deployment is within time range
		deployTime := time.Time{}
		if d.CreateTime != "" {
			if t, err := time.Parse(time.RFC3339, d.CreateTime); err == nil {
				deployTime = t
			}
		}

		if !deployTime.IsZero() && deployTime.Before(cutoff) {
			continue
		}

		metrics.Total++

		switch d.Status.State {
		case apps.AppDeploymentStateSucceeded:
			succeeded++
		case apps.AppDeploymentStateFailed:
			failed++
		case apps.AppDeploymentStateCancelled, apps.AppDeploymentStateInProgress:
			// Don't count cancelled or in-progress deployments
		}

		if deployTime.After(lastDeployed) {
			lastDeployed = deployTime
		}
	}

	metrics.Succeeded = succeeded
	metrics.Failed = failed

	if metrics.Total > 0 {
		metrics.SuccessRate = float64(succeeded) / float64(metrics.Total) * 100
	}

	if !lastDeployed.IsZero() {
		metrics.LastDeployed = lastDeployed.Format(time.RFC3339)
	}

	return metrics
}

func parseTimeRange(timeRange string) time.Duration {
	timeRange = strings.ToLower(strings.TrimSpace(timeRange))

	switch timeRange {
	case "1h":
		return time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}
