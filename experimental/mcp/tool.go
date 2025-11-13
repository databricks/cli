package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/experimental/mcp/tools"
	"github.com/spf13/cobra"
)

func newToolCmd() *cobra.Command {
	var jsonData string
	cmd := &cobra.Command{
		Use:    "tool <tool_name>",
		Short:  "Run a specific MCP tool directly, for testing purposes",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var result string
			var err error
			switch args[0] {
			case "invoke_databricks_cli":
				result, err = unmarshal(ctx, []byte(jsonData), tools.InvokeDatabricksCLI)
			case "init_project":
				result, err = unmarshal(ctx, []byte(jsonData), tools.InitProject)
			case "add_project_resource":
				result, err = unmarshal(ctx, []byte(jsonData), tools.AddProjectResource)
			case "analyze_project":
				result, err = unmarshal(ctx, []byte(jsonData), tools.AnalyzeProject)
			case "explore":
				result, err = tools.ExploreTool.Handler(ctx, make(map[string]any))
			default:
				return fmt.Errorf("unknown tool: %s", args[0])
			}

			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout, result)
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonData, "json", "", "JSON arguments for the tool")
	cmd.MarkFlagRequired("json")
	return cmd
}

func unmarshal[T any](ctx context.Context, data []byte, fn func(context.Context, T) (string, error)) (string, error) {
	var args T
	if err := json.Unmarshal(data, &args); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}
	return fn(ctx, args)
}
