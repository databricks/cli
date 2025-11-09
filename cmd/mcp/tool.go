package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/mcp/tools"
)

// runTool executes a specific MCP tool for acceptance testing.
// This is a hidden command accessed via --tool and --config-file flags.
func runTool(ctx context.Context, toolName, configFile string) error {
	if configFile == "" {
		return errors.New("--config-file is required when using --tool")
	}

	// Read config file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Execute tool based on name
	var result string
	switch toolName {
	case "invoke_databricks_cli":
		result, err = runInvokeDatabricksCLITool(ctx, configData)
	case "init_project":
		result, err = runInitProjectTool(ctx, configData)
	case "extend_project":
		result, err = runExtendProjectTool(ctx, configData)
	case "analyze_project":
		result, err = runAnalyzeProjectTool(ctx, configData)
	default:
		return fmt.Errorf("unknown tool: %s. Valid tools: invoke_databricks_cli, init_project, extend_project, analyze_project", toolName)
	}

	if err != nil {
		return err
	}

	// Output result to stdout
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

func runExtendProjectTool(ctx context.Context, configData []byte) (string, error) {
	var args tools.ExtendProjectArgs
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

	return tools.ExtendProject(ctx, args)
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
