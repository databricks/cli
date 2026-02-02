package mcp

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/providers/sdkdocs"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newQuerySDKDocsCmd() *cobra.Command {
	var category string
	var service string
	var limit int

	cmd := &cobra.Command{
		Use:   "query-sdk-docs QUERY",
		Short: "Search Databricks SDK documentation",
		Long: `Search Databricks SDK documentation for methods, types, and enums.

Use this tool to find:
- API methods: "how to create a job", "list clusters", "run pipeline"
- Type definitions: "JobSettings fields", "ClusterSpec parameters"
- Enums: "run lifecycle states", "cluster state values"

The search returns method signatures, descriptions, and parameter information.`,
		Example: `  # Search for job creation methods
  databricks experimental aitools tools query-sdk-docs "create job"

  # Search only methods in the jobs service
  databricks experimental aitools tools query-sdk-docs "list" --service jobs --category methods

  # Find type definitions
  databricks experimental aitools tools query-sdk-docs "JobSettings" --category types

  # Find enum values
  databricks experimental aitools tools query-sdk-docs "lifecycle state" --category enums`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			query := args[0]

			if query == "" {
				return fmt.Errorf("query is required")
			}

			// Load the SDK docs index
			index, err := sdkdocs.LoadIndex()
			if err != nil {
				return fmt.Errorf("failed to load SDK docs index: %w", err)
			}

			// Search the index
			results := index.Search(sdkdocs.SearchOptions{
				Query:    query,
				Category: category,
				Service:  service,
				Limit:    limit,
			})

			if len(results) == 0 {
				cmdio.LogString(ctx, "No results found for query: "+query)
				return nil
			}

			// Format results
			output := formatSDKDocsResults(results)
			cmdio.LogString(ctx, output)
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by category: methods, types, enums, services")
	cmd.Flags().StringVar(&service, "service", "", "Filter by service: jobs, clusters, pipelines, etc.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of results (default 10, max 50)")

	return cmd
}

func formatSDKDocsResults(results []sdkdocs.SearchResult) string {
	var sb strings.Builder

	sb.WriteString("## SDK Documentation Results\n\n")

	for _, r := range results {
		switch r.Type {
		case "method":
			sb.WriteString(fmt.Sprintf("### Method: %s.%s\n\n", r.Service, r.Name))
			if r.Signature != "" {
				sb.WriteString("**Signature:**\n```go\n")
				sb.WriteString(r.Signature)
				sb.WriteString("\n```\n\n")
			}
			if r.Description != "" {
				sb.WriteString("**Description:**\n")
				sb.WriteString(r.Description)
				sb.WriteString("\n\n")
			}

		case "type":
			sb.WriteString(fmt.Sprintf("### Type: %s\n\n", r.Name))
			if r.Description != "" {
				sb.WriteString("**Description:**\n")
				sb.WriteString(r.Description)
				sb.WriteString("\n\n")
			}
			if len(r.Fields) > 0 {
				sb.WriteString("**Fields:**\n")
				for _, field := range r.Fields {
					sb.WriteString(fmt.Sprintf("- `%s` (%s): %s\n", field.Name, field.Type, field.Description))
				}
				sb.WriteString("\n")
			}

		case "enum":
			sb.WriteString(fmt.Sprintf("### Enum: %s\n\n", r.Name))
			if r.Description != "" {
				sb.WriteString("**Description:**\n")
				sb.WriteString(r.Description)
				sb.WriteString("\n\n")
			}
			if len(r.Values) > 0 {
				sb.WriteString("**Values:** ")
				sb.WriteString(strings.Join(r.Values, ", "))
				sb.WriteString("\n\n")
			}

		case "service":
			sb.WriteString(fmt.Sprintf("### Service: %s\n\n", r.Name))
			if r.Description != "" {
				sb.WriteString("**Description:**\n")
				sb.WriteString(r.Description)
				sb.WriteString("\n\n")
			}
		}

		sb.WriteString("---\n\n")
	}

	return sb.String()
}
