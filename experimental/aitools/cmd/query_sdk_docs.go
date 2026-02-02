package mcp

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/providers/sdkdocs"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newQuerySDKDocsCmd() *cobra.Command {
	var category, service string
	var limit int

	cmd := &cobra.Command{
		Use:     "query-sdk-docs QUERY",
		Short:   "Search Databricks SDK documentation",
		Example: `  databricks experimental aitools tools query-sdk-docs "create job"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			index, err := sdkdocs.LoadIndex()
			if err != nil {
				return err
			}

			results := index.Search(sdkdocs.SearchOptions{
				Query:    args[0],
				Category: category,
				Service:  service,
				Limit:    limit,
			})

			if len(results) == 0 {
				cmdio.LogString(ctx, "No results found")
				return nil
			}

			cmdio.LogString(ctx, formatResults(results))
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter: methods, types, enums, services")
	cmd.Flags().StringVar(&service, "service", "", "Filter by service: jobs, compute, pipelines, etc.")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results (default 10)")

	return cmd
}

func formatResults(results []sdkdocs.SearchResult) string {
	var sb strings.Builder
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("### %s: %s\n", r.Type, r.Name))
		if r.Signature != "" {
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n", r.Signature))
		}
		if r.Description != "" {
			sb.WriteString(r.Description + "\n")
		}
		if len(r.Fields) > 0 {
			for _, f := range r.Fields {
				sb.WriteString(fmt.Sprintf("  - %s (%s)\n", f.Name, f.Type))
			}
		}
		if len(r.Values) > 0 {
			sb.WriteString("Values: " + strings.Join(r.Values, ", ") + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
