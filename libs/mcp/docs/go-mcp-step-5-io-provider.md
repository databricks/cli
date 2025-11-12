# Step 5: I/O Provider

## Overview
Implement the I/O provider for scaffolding projects from templates and validating them in sandboxed environments.

## Tasks

### 5.1 Design Template System

**pkg/templates/template.go:**

```go
type Template interface {
    Name() string
    Description() string
    Files() (map[string]string, error)  // path -> content
}

type EmbeddedTemplate struct {
    name        string
    description string
    fsys        embed.FS
    root        string
}

func NewEmbeddedTemplate(name, desc string, fsys embed.FS, root string) *EmbeddedTemplate {
    return &EmbeddedTemplate{
        name:        name,
        description: desc,
        fsys:        fsys,
        root:        root,
    }
}

func (t *EmbeddedTemplate) Name() string {
    return t.name
}

func (t *EmbeddedTemplate) Description() string {
    return t.description
}

func (t *EmbeddedTemplate) Files() (map[string]string, error) {
    files := make(map[string]string)

    err := fs.WalkDir(t.fsys, t.root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        if d.IsDir() {
            return nil
        }

        // Read file content
        content, err := fs.ReadFile(t.fsys, path)
        if err != nil {
            return err
        }

        // Remove root prefix from path
        relativePath := strings.TrimPrefix(path, t.root+"/")
        files[relativePath] = string(content)

        return nil
    })

    return files, err
}
```

### 5.2 Embed Default Templates

**internal/templates/embed.go:**

```go
//go:embed trpc/*
var trpcFS embed.FS

func GetTRPCTemplate() templates.Template {
    return templates.NewEmbeddedTemplate(
        "TRPC",
        "Modern full-stack template with tRPC, TypeScript, and React",
        trpcFS,
        "trpc",
    )
}

// Create actual template files in internal/templates/trpc/
// - package.json
// - tsconfig.json
// - src/server/index.ts
// - src/client/index.tsx
// - etc.
```

Create minimal template structure:
```
internal/templates/trpc/
├── package.json
├── tsconfig.json
├── .gitignore
├── src/
│   ├── server/
│   │   └── index.ts
│   └── client/
│       └── index.tsx
└── README.md
```

### 5.3 Implement Scaffold Operation

**pkg/providers/io/scaffold.go:**

```go
type ScaffoldArgs struct {
    WorkDir       string `json:"work_dir"`
    ForceRewrite  bool   `json:"force_rewrite,omitempty"`
}

type ScaffoldResult struct {
    FilesCopied         int    `json:"files_copied"`
    WorkDir             string `json:"work_dir"`
    TemplateName        string `json:"template_name"`
    TemplateDescription string `json:"template_description"`
}

func (p *Provider) Scaffold(ctx context.Context, args *ScaffoldArgs) (*ScaffoldResult, error) {
    // Validate work directory
    workDir, err := filepath.Abs(args.WorkDir)
    if err != nil {
        return nil, fmt.Errorf("invalid work directory: %w", err)
    }

    // Check if directory exists
    if stat, err := os.Stat(workDir); err == nil {
        if !stat.IsDir() {
            return nil, errors.New("work_dir exists but is not a directory")
        }

        // Check if empty
        entries, err := os.ReadDir(workDir)
        if err != nil {
            return nil, err
        }

        if len(entries) > 0 && !args.ForceRewrite {
            return nil, errors.New("work_dir is not empty (use force_rewrite to overwrite)")
        }

        // Clear directory if force_rewrite
        if args.ForceRewrite {
            if err := os.RemoveAll(workDir); err != nil {
                return nil, fmt.Errorf("failed to clear directory: %w", err)
            }
        }
    }

    // Create directory
    if err := os.MkdirAll(workDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create directory: %w", err)
    }

    // Get template
    template := p.getTemplate()
    files, err := template.Files()
    if err != nil {
        return nil, fmt.Errorf("failed to read template: %w", err)
    }

    // Copy files
    filesCopied := 0
    for path, content := range files {
        targetPath := filepath.Join(workDir, path)

        // Create parent directories
        if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
            return nil, fmt.Errorf("failed to create directory for %s: %w", path, err)
        }

        // Write file
        if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
            return nil, fmt.Errorf("failed to write %s: %w", path, err)
        }

        filesCopied++
    }

    p.logger.Info("scaffolded project",
        "template", template.Name(),
        "work_dir", workDir,
        "files", filesCopied)

    return &ScaffoldResult{
        FilesCopied:         filesCopied,
        WorkDir:             workDir,
        TemplateName:        template.Name(),
        TemplateDescription: template.Description(),
    }, nil
}
```

### 5.4 Implement Validation

**pkg/providers/io/validate.go:**

```go
type ValidateArgs struct {
    WorkDir string `json:"work_dir"`
}

type ValidateResult struct {
    Success bool              `json:"success"`
    Message string            `json:"message"`
    Details *ValidationDetail `json:"details,omitempty"`
}

type ValidationDetail struct {
    ExitCode int    `json:"exit_code"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
}

func (p *Provider) Validate(ctx context.Context, args *ValidateArgs) (*ValidateResult, error) {
    // Validate work directory exists
    workDir, err := filepath.Abs(args.WorkDir)
    if err != nil {
        return nil, fmt.Errorf("invalid work directory: %w", err)
    }

    if _, err := os.Stat(workDir); os.IsNotExist(err) {
        return nil, errors.New("work directory does not exist")
    }

    // Get validation config
    if p.config == nil || p.config.Validation == nil {
        return &ValidateResult{
            Success: true,
            Message: "No validation configured - skipping",
        }, nil
    }

    valConfig := p.config.Validation

    // Create sandbox
    sb, err := sandbox.New(sandbox.TypeLocal, sandbox.WithBaseDir(workDir))
    if err != nil {
        return nil, fmt.Errorf("failed to create sandbox: %w", err)
    }
    defer sb.Close()

    // Execute validation command
    p.logger.Info("running validation", "command", valConfig.Command)

    result, err := sb.Exec(ctx, valConfig.Command)
    if err != nil {
        return nil, fmt.Errorf("validation execution failed: %w", err)
    }

    success := result.ExitCode == 0
    message := "Validation passed"
    if !success {
        message = "Validation failed"
    }

    return &ValidateResult{
        Success: success,
        Message: message,
        Details: &ValidationDetail{
            ExitCode: result.ExitCode,
            Stdout:   result.Stdout,
            Stderr:   result.Stderr,
        },
    }, nil
}
```

### 5.5 Create I/O Provider

**pkg/providers/io/provider.go:**

```go
type Provider struct {
    config   *config.IOConfig
    registry *mcp.Registry
    logger   *slog.Logger
}

func NewProvider(cfg *config.IOConfig, logger *slog.Logger) (*Provider, error) {
    p := &Provider{
        config:   cfg,
        registry: mcp.NewRegistry(),
        logger:   logger,
    }

    if err := p.registerTools(); err != nil {
        return nil, err
    }

    return p, nil
}

func (p *Provider) registerTools() error {
    tools := []struct {
        tool mcp.Tool
        fn   mcp.ToolFunc
    }{
        {
            tool: mcp.Tool{
                Name:        "scaffold_data_app",
                Description: "Scaffold a new data application from template",
                InputSchema: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "work_dir": map[string]any{
                            "type":        "string",
                            "description": "Absolute path to the work directory",
                        },
                        "force_rewrite": map[string]any{
                            "type":        "boolean",
                            "description": "Overwrite existing files if directory is not empty",
                            "default":     false,
                        },
                    },
                    "required": []string{"work_dir"},
                },
            },
            fn: p.handleScaffold,
        },
        {
            tool: mcp.Tool{
                Name:        "validate_data_app",
                Description: "Validate a data application by running tests",
                InputSchema: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "work_dir": map[string]any{
                            "type":        "string",
                            "description": "Absolute path to the work directory",
                        },
                    },
                    "required": []string{"work_dir"},
                },
            },
            fn: p.handleValidate,
        },
    }

    for _, t := range tools {
        if err := p.registry.Register(t.tool, t.fn); err != nil {
            return err
        }
    }

    return nil
}

func (p *Provider) handleScaffold(ctx context.Context, params json.RawMessage) (*mcp.ToolResult, error) {
    var args ScaffoldArgs
    if err := json.Unmarshal(params, &args); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }

    result, err := p.Scaffold(ctx, &args)
    if err != nil {
        return nil, err
    }

    text := formatScaffoldResult(result)
    return &mcp.ToolResult{
        Content: []mcp.Content{{Type: "text", Text: text}},
    }, nil
}

func (p *Provider) handleValidate(ctx context.Context, params json.RawMessage) (*mcp.ToolResult, error) {
    var args ValidateArgs
    if err := json.Unmarshal(params, &args); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }

    result, err := p.Validate(ctx, &args)
    if err != nil {
        return nil, err
    }

    text := formatValidateResult(result)
    return &mcp.ToolResult{
        Content: []mcp.Content{{Type: "text", Text: text}},
        IsError: !result.Success,
    }, nil
}

func (p *Provider) getTemplate() templates.Template {
    if p.config != nil && p.config.Template != nil {
        // Handle custom template if specified
        if p.config.Template.Path != "" {
            // Load from filesystem
        }
    }

    // Default to TRPC template
    return templates.GetTRPCTemplate()
}

func (p *Provider) ListTools(ctx context.Context) ([]mcp.Tool, error) {
    return p.registry.ListTools(), nil
}

func (p *Provider) CallTool(ctx context.Context, name string, params json.RawMessage) (*mcp.ToolResult, error) {
    return p.registry.Call(ctx, name, params)
}
```

### 5.6 Add Result Formatting

**pkg/providers/io/format.go:**

```go
func formatScaffoldResult(result *ScaffoldResult) string {
    return fmt.Sprintf(
        "Successfully scaffolded %s template to %s\n\n"+
            "Files copied: %d\n\n"+
            "Template: %s\n\n"+
            "%s",
        result.TemplateName,
        result.WorkDir,
        result.FilesCopied,
        result.TemplateName,
        result.TemplateDescription,
    )
}

func formatValidateResult(result *ValidateResult) string {
    if result.Success {
        return fmt.Sprintf("✓ %s", result.Message)
    }

    if result.Details == nil {
        return fmt.Sprintf("✗ %s", result.Message)
    }

    return fmt.Sprintf(
        "✗ %s\n\nExit code: %d\n\nStdout:\n%s\n\nStderr:\n%s",
        result.Message,
        result.Details.ExitCode,
        result.Details.Stdout,
        result.Details.Stderr,
    )
}
```

### 5.7 Write Tests

**pkg/providers/io/scaffold_test.go:**

```go
func TestProvider_Scaffold(t *testing.T) {
    tests := []struct {
        name      string
        args      *ScaffoldArgs
        setup     func(*testing.T) string  // Returns temp dir
        wantErr   bool
        validate  func(*testing.T, string, *ScaffoldResult)
    }{
        {
            name: "scaffold to empty directory",
            args: &ScaffoldArgs{},  // WorkDir set in test
            setup: func(t *testing.T) string {
                return t.TempDir()
            },
            wantErr: false,
            validate: func(t *testing.T, dir string, result *ScaffoldResult) {
                // Check files exist
                assert.FileExists(t, filepath.Join(dir, "package.json"))
                assert.Greater(t, result.FilesCopied, 0)
            },
        },
        {
            name: "scaffold to non-empty directory without force",
            args: &ScaffoldArgs{
                ForceRewrite: false,
            },
            setup: func(t *testing.T) string {
                dir := t.TempDir()
                // Create existing file
                os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("test"), 0644)
                return dir
            },
            wantErr: true,
        },
        {
            name: "scaffold with force rewrite",
            args: &ScaffoldArgs{
                ForceRewrite: true,
            },
            setup: func(t *testing.T) string {
                dir := t.TempDir()
                os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("test"), 0644)
                return dir
            },
            wantErr: false,
            validate: func(t *testing.T, dir string, result *ScaffoldResult) {
                // Old file should be gone
                assert.NoFileExists(t, filepath.Join(dir, "existing.txt"))
                // New files exist
                assert.FileExists(t, filepath.Join(dir, "package.json"))
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := tt.setup(t)
            tt.args.WorkDir = dir

            p, err := NewProvider(nil, slog.Default())
            require.NoError(t, err)

            result, err := p.Scaffold(context.Background(), tt.args)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                if tt.validate != nil {
                    tt.validate(t, dir, result)
                }
            }
        })
    }
}
```

**pkg/providers/io/validate_test.go:**

```go
func TestProvider_Validate(t *testing.T)
func TestProvider_ValidateNonExistentDir(t *testing.T)
func TestProvider_ValidateWithCustomCommand(t *testing.T)
```

## Acceptance Criteria

- [ ] Template system can read embedded files
- [ ] Default TRPC template embedded
- [ ] Scaffold operation copies files correctly
- [ ] Force rewrite clears existing directory
- [ ] Validation runs in sandbox
- [ ] Validation captures output and exit code
- [ ] Tools registered with MCP
- [ ] All unit tests pass
- [ ] Can scaffold and validate real project

## Testing Commands

```bash
# Unit tests
go test ./pkg/providers/io/...
go test ./pkg/templates/...

# Integration test
go test ./test/integration/io_test.go
```

## Next Steps

Proceed to Step 6: Workspace Provider once all acceptance criteria are met.
