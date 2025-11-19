package workspace

import (
	"context"
	"encoding/json"
	"fmt"

	mcp "github.com/databricks/cli/experimental/apps-mcp/lib"
	mcpsdk "github.com/databricks/cli/experimental/apps-mcp/lib/mcp"
	"github.com/databricks/cli/experimental/apps-mcp/lib/providers"
	"github.com/databricks/cli/experimental/apps-mcp/lib/session"
	"github.com/databricks/cli/libs/log"
)

func init() {
	providers.Register("workspace", func(ctx context.Context, cfg *mcp.Config, sess *session.Session) (providers.Provider, error) {
		return NewProvider(ctx, sess)
	}, providers.ProviderConfig{
		EnabledWhen: func(cfg *mcp.Config) bool {
			return cfg.WithWorkspaceTools
		},
	})
}

// Provider implements the workspace provider for file operations
type Provider struct {
	session *session.Session
	ctx     context.Context
}

// NewProvider creates a new workspace provider
func NewProvider(ctx context.Context, sess *session.Session) (*Provider, error) {
	return &Provider{
		session: sess,
		ctx:     ctx,
	}, nil
}

// Name returns the name of the provider.
func (p *Provider) Name() string {
	return "workspace"
}

// getWorkDir retrieves the working directory from the session via context
func (p *Provider) getWorkDir(ctx context.Context) (string, error) {
	workDir, err := session.GetWorkDir(ctx)
	if err != nil {
		return "", fmt.Errorf(
			"workspace directory not set - please run scaffold_data_app first to initialize your project: %w",
			err,
		)
	}
	return workDir, nil
}

// RegisterTools registers all workspace tools with the MCP server
func (p *Provider) RegisterTools(server *mcpsdk.Server) error {
	log.Info(p.ctx, "Registering workspace tools")

	// Register read_file
	type ReadFileInput struct {
		FilePath string `json:"file_path" jsonschema:"required" jsonschema_description:"Path to file relative to workspace"`
		Offset   int    `json:"offset,omitempty" jsonschema_description:"Line number to start reading (1-indexed)"`
		Limit    int    `json:"limit,omitempty" jsonschema_description:"Number of lines to read"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "read_file",
			Description: "Read file contents with line numbers. Default: reads up to 2000 lines from beginning. Lines >2000 chars truncated.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args ReadFileInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "read_file called: file_path=%s", args.FilePath)

			readArgs := &ReadFileArgs{
				FilePath: args.FilePath,
				Offset:   args.Offset,
				Limit:    args.Limit,
			}

			content, err := p.ReadFile(ctx, readArgs)
			if err != nil {
				return nil, nil, err
			}

			return mcpsdk.CreateNewTextContentResult(content), nil, nil
		}),
	)

	// Register write_file
	type WriteFileInput struct {
		FilePath string `json:"file_path" jsonschema:"required" jsonschema_description:"Path to file relative to workspace"`
		Content  string `json:"content" jsonschema:"required" jsonschema_description:"Content to write"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "write_file",
			Description: "Write content to a file",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args WriteFileInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "write_file called: file_path=%s", args.FilePath)

			writeArgs := &WriteFileArgs{
				FilePath: args.FilePath,
				Content:  args.Content,
			}

			err := p.WriteFile(ctx, writeArgs)
			if err != nil {
				return nil, nil, err
			}

			return mcpsdk.CreateNewTextContentResult("File written successfully: " + args.FilePath), nil, nil
		}),
	)

	// Register edit_file
	type EditFileInput struct {
		FilePath  string `json:"file_path" jsonschema:"required" jsonschema_description:"Path to file relative to workspace"`
		OldString string `json:"old_string" jsonschema:"required" jsonschema_description:"String to replace (must be unique)"`
		NewString string `json:"new_string" jsonschema:"required" jsonschema_description:"Replacement string"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "edit_file",
			Description: "Edit file by replacing old_string with new_string. Fails if old_string not unique unless replace_all=true.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args EditFileInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "edit_file called: file_path=%s", args.FilePath)

			editArgs := &EditFileArgs{
				FilePath:  args.FilePath,
				OldString: args.OldString,
				NewString: args.NewString,
			}

			err := p.EditFile(ctx, editArgs)
			if err != nil {
				return nil, nil, err
			}

			return mcpsdk.CreateNewTextContentResult("File edited successfully: " + args.FilePath), nil, nil
		}),
	)

	// Register bash
	type BashInput struct {
		Command string `json:"command" jsonschema:"required" jsonschema_description:"Bash command to execute"`
		Timeout int    `json:"timeout,omitempty" jsonschema_description:"Timeout in seconds (default 120)"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "bash",
			Description: "Execute bash command in workspace directory. Use for terminal operations (npm, git, etc). Output truncated at 30000 chars.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args BashInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "bash called: command=%s", args.Command)

			bashArgs := &BashArgs{
				Command: args.Command,
				Timeout: args.Timeout,
			}

			result, err := p.Bash(ctx, bashArgs)
			if err != nil {
				return nil, nil, err
			}

			// Format result as JSON
			resultJSON, _ := json.Marshal(result)

			return mcpsdk.CreateNewTextContentResult(string(resultJSON)), nil, nil
		}),
	)

	// Register grep
	type GrepInput struct {
		Pattern    string `json:"pattern" jsonschema:"required" jsonschema_description:"Regular expression pattern to search for"`
		Path       string `json:"path,omitempty" jsonschema_description:"Limit search to specific path"`
		IgnoreCase bool   `json:"ignore_case,omitempty" jsonschema_description:"Case insensitive search"`
		MaxResults int    `json:"max_results,omitempty" jsonschema_description:"Maximum number of results (default 100)"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "grep",
			Description: "Search file contents with regex. Returns file:line:content by default. Limit results with head_limit.",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args GrepInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "grep called: pattern=%s", args.Pattern)

			grepArgs := &GrepArgs{
				Pattern:    args.Pattern,
				Path:       args.Path,
				IgnoreCase: args.IgnoreCase,
				MaxResults: args.MaxResults,
			}

			result, err := p.Grep(ctx, grepArgs)
			if err != nil {
				return nil, nil, err
			}

			// Format result as JSON
			resultJSON, _ := json.Marshal(result)

			return mcpsdk.CreateNewTextContentResult(string(resultJSON)), nil, nil
		}),
	)

	// Register glob
	type GlobInput struct {
		Pattern string `json:"pattern" jsonschema:"required" jsonschema_description:"File pattern to match (e.g., '*.go', 'src/**/*.ts')"`
	}

	mcpsdk.AddTool(server,
		&mcpsdk.Tool{
			Name:        "glob",
			Description: "Find files matching a glob pattern",
		},
		session.WrapToolHandler(p.session, func(ctx context.Context, req *mcpsdk.CallToolRequest, args GlobInput) (*mcpsdk.CallToolResult, any, error) {
			log.Debugf(ctx, "glob called: pattern=%s", args.Pattern)

			globArgs := &GlobArgs{
				Pattern: args.Pattern,
			}

			result, err := p.Glob(ctx, globArgs)
			if err != nil {
				return nil, nil, err
			}

			// Format result as JSON
			resultJSON, _ := json.Marshal(result)

			return mcpsdk.CreateNewTextContentResult(string(resultJSON)), nil, nil
		}),
	)
	return nil
}
