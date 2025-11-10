package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/mcp/tools"
	"github.com/spf13/cobra"
)

func newToolCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:    "tool <tool_name>",
		Short:  "Run a specific MCP tool for testing (hidden)",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolName := args[0]
			return runTool(cmd.Context(), toolName, configFile)
		},
	}

	cmd.Flags().StringVar(&configFile, "config-file", "", "JSON config file for tool arguments")
	cmd.MarkFlagRequired("config-file")

	return cmd
}

// runTool executes a specific MCP tool for acceptance testing.
// This is a hidden command accessed via 'databricks mcp tool <tool_name> --config-file <file>'.
func runTool(ctx context.Context, toolName, configFile string) error {
	if configFile == "" {
		return errors.New("--config-file is required")
	}

	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var result string
	switch toolName {
	case "invoke_databricks_cli":
		result, err = runInvokeDatabricksCLITool(ctx, configData)
	case "init_project":
		result, err = runInitProjectTool(ctx, configData)
	case "add_project_resource":
		result, err = runAddProjectResourceTool(ctx, configData)
	case "analyze_project":
		result, err = runAnalyzeProjectTool(ctx, configData)
	default:
		return fmt.Errorf("unknown tool: %s. Valid tools: invoke_databricks_cli, init_project, add_project_resource, analyze_project", toolName)
	}

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, result)
	return err
}

func runInitProjectTool(ctx context.Context, configData []byte) (string, error) {
	var args tools.InitProjectArgs
	if err := json.Unmarshal(configData, &args); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	return tools.InitProject(ctx, args)
}

func runAddProjectResourceTool(ctx context.Context, configData []byte) (string, error) {
	var args tools.AddProjectResourceArgs
	if err := json.Unmarshal(configData, &args); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	// If project_path is empty or ".", use current working directory
	if args.ProjectPath == "" || args.ProjectPath == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		args.ProjectPath = cwd
	}

	return tools.AddProjectResource(ctx, args)
}

func runAnalyzeProjectTool(ctx context.Context, configData []byte) (string, error) {
	var args tools.AnalyzeProjectArgs
	if err := json.Unmarshal(configData, &args); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	// If project_path is empty or ".", use current working directory
	if args.ProjectPath == "" || args.ProjectPath == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		args.ProjectPath = cwd
	}

	return tools.AnalyzeProject(ctx, args)
}

func runInvokeDatabricksCLITool(ctx context.Context, configData []byte) (string, error) {
	var args tools.InvokeDatabricksCLIArgs
	if err := json.Unmarshal(configData, &args); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	return tools.InvokeDatabricksCLI(ctx, args)
}
