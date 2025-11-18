package tools

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/exec"
)

// InvokeDatabricksCLITool runs databricks CLI commands via MCP.
var InvokeDatabricksCLITool = Tool{
	Definition: ToolDefinition{
		Name:        "invoke_databricks_cli",
		Description: "Run any Databricks CLI command. Use this tool whenever you need to run databricks CLI commands like 'bundle deploy', 'bundle validate', 'bundle run', 'auth login', etc. The reason this tool exists (instead of invoking the databricks CLI directly) is to make it easier for users to allow-list commands.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The full Databricks CLI command to run, e.g. 'bundle deploy' or 'bundle validate'. Do not include the 'databricks' prefix.",
				},
				"working_directory": map[string]any{
					"type":        "string",
					"description": "Optional. The directory to run the command in. Defaults to the current directory.",
				},
			},
			"required": []string{"command"},
		},
	},
	Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var typedArgs invokeDatabricksCLIArgs
		if err := UnmarshalArgs(args, &typedArgs); err != nil {
			return "", err
		}
		return InvokeDatabricksCLI(ctx, typedArgs)
	},
}

type invokeDatabricksCLIArgs struct {
	Command          string `json:"command"`
	WorkingDirectory string `json:"working_directory,omitempty"`
}

// InvokeDatabricksCLI runs a Databricks CLI command and returns the output.
func InvokeDatabricksCLI(ctx context.Context, args invokeDatabricksCLIArgs) (string, error) {
	if args.Command == "" {
		return "", errors.New("command is required")
	}

	workDir := args.WorkingDirectory
	if workDir == "" {
		workDir = "."
	}

	executor, err := exec.NewCommandExecutor(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to create command executor: %w", err)
	}

	executor.WithEnv(append(os.Environ(), "DATABRICKS_CLI_UPSTREAM=aitools"))

	fullCommand := fmt.Sprintf(`"%s" %s`, GetCLIPath(), args.Command)
	output, err := executor.Exec(ctx, fullCommand)

	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\n\nCommand failed with error: %v", err)
		return result, nil
	}

	return result, nil
}
