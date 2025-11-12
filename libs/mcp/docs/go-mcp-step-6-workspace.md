# Step 6: Workspace Provider

## Overview
Implement workspace tools for file operations, bash execution, and code search within a project directory. Similar to Claude Code's base toolkit.

## Tasks

### 6.1 Create Workspace Provider Structure

**pkg/providers/workspace/provider.go:**

```go
type Provider struct {
    session  *session.Session
    registry *mcp.Registry
    logger   *slog.Logger
}

func NewProvider(sess *session.Session, logger *slog.Logger) (*Provider, error) {
    p := &Provider{
        session:  sess,
        registry: mcp.NewRegistry(),
        logger:   logger,
    }

    if err := p.registerTools(); err != nil {
        return nil, err
    }

    return p, nil
}

func (p *Provider) getWorkDir() (string, error) {
    workDir, err := p.session.GetWorkDir()
    if err != nil {
        return "", fmt.Errorf(
            "workspace directory not set. "+
                "Please run scaffold_data_app first to initialize your project",
        )
    }
    return workDir, nil
}
```

### 6.2 Implement File Operations

**pkg/providers/workspace/files.go:**

```go
type ReadFileArgs struct {
    FilePath string `json:"file_path"`
    Offset   int    `json:"offset,omitempty"`   // Line number to start (1-indexed)
    Limit    int    `json:"limit,omitempty"`    // Number of lines to read
}

func (p *Provider) ReadFile(ctx context.Context, args *ReadFileArgs) (string, error) {
    workDir, err := p.getWorkDir()
    if err != nil {
        return "", err
    }

    // Validate path
    fullPath, err := validatePath(workDir, args.FilePath)
    if err != nil {
        return "", err
    }

    // Read file
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return "", fmt.Errorf("failed to read file: %w", err)
    }

    // Apply line offset and limit if specified
    if args.Offset > 0 || args.Limit > 0 {
        content = applyLineRange(content, args.Offset, args.Limit)
    }

    return string(content), nil
}

type WriteFileArgs struct {
    FilePath string `json:"file_path"`
    Content  string `json:"content"`
}

func (p *Provider) WriteFile(ctx context.Context, args *WriteFileArgs) error {
    workDir, err := p.getWorkDir()
    if err != nil {
        return err
    }

    // Validate path
    fullPath, err := validatePath(workDir, args.FilePath)
    if err != nil {
        return err
    }

    // Create parent directories
    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        return fmt.Errorf("failed to create parent directory: %w", err)
    }

    // Write file atomically
    tmpFile := fullPath + ".tmp"
    if err := os.WriteFile(tmpFile, []byte(args.Content), 0644); err != nil {
        return fmt.Errorf("failed to write temp file: %w", err)
    }

    if err := os.Rename(tmpFile, fullPath); err != nil {
        os.Remove(tmpFile)
        return fmt.Errorf("failed to rename file: %w", err)
    }

    return nil
}

type EditFileArgs struct {
    FilePath  string `json:"file_path"`
    OldString string `json:"old_string"`
    NewString string `json:"new_string"`
}

func (p *Provider) EditFile(ctx context.Context, args *EditFileArgs) error {
    workDir, err := p.getWorkDir()
    if err != nil {
        return err
    }

    // Validate path
    fullPath, err := validatePath(workDir, args.FilePath)
    if err != nil {
        return err
    }

    // Read current content
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }

    // Check if old_string exists
    contentStr := string(content)
    if !strings.Contains(contentStr, args.OldString) {
        return errors.New("old_string not found in file")
    }

    // Count occurrences
    count := strings.Count(contentStr, args.OldString)
    if count > 1 {
        return fmt.Errorf("old_string appears %d times, must be unique", count)
    }

    // Replace
    newContent := strings.Replace(contentStr, args.OldString, args.NewString, 1)

    // Write back
    if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
        return fmt.Errorf("failed to write file: %w", err)
    }

    return nil
}

// Helper function to apply line range
func applyLineRange(content []byte, offset, limit int) []byte {
    lines := bytes.Split(content, []byte("\n"))

    // Adjust for 1-indexed
    if offset > 0 {
        offset--
    }

    // Apply offset
    if offset > 0 {
        if offset >= len(lines) {
            return []byte{}
        }
        lines = lines[offset:]
    }

    // Apply limit
    if limit > 0 && limit < len(lines) {
        lines = lines[:limit]
    }

    return bytes.Join(lines, []byte("\n"))
}
```

### 6.3 Implement Bash Execution

**pkg/providers/workspace/bash.go:**

```go
type BashArgs struct {
    Command string `json:"command"`
    Timeout int    `json:"timeout,omitempty"`  // Seconds, default 120
}

type BashResult struct {
    ExitCode int    `json:"exit_code"`
    Stdout   string `json:"stdout"`
    Stderr   string `json:"stderr"`
}

func (p *Provider) Bash(ctx context.Context, args *BashArgs) (*BashResult, error) {
    workDir, err := p.getWorkDir()
    if err != nil {
        return nil, err
    }

    // Set timeout
    timeout := time.Duration(args.Timeout) * time.Second
    if args.Timeout == 0 {
        timeout = 120 * time.Second
    }

    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Execute command
    cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
    cmd.Dir = workDir

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err = cmd.Run()

    exitCode := 0
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            exitCode = exitErr.ExitCode()
        } else if ctx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("command timed out after %d seconds", args.Timeout)
        } else {
            return nil, fmt.Errorf("failed to execute command: %w", err)
        }
    }

    return &BashResult{
        ExitCode: exitCode,
        Stdout:   stdout.String(),
        Stderr:   stderr.String(),
    }, nil
}
```

### 6.4 Implement Grep

**pkg/providers/workspace/grep.go:**

```go
type GrepArgs struct {
    Pattern    string `json:"pattern"`
    Path       string `json:"path,omitempty"`       // Limit to specific path
    IgnoreCase bool   `json:"ignore_case,omitempty"`
    MaxResults int    `json:"max_results,omitempty"` // Default 100
}

type GrepResult struct {
    Matches []GrepMatch `json:"matches"`
    Total   int         `json:"total"`
}

type GrepMatch struct {
    File    string `json:"file"`
    Line    int    `json:"line"`
    Content string `json:"content"`
}

func (p *Provider) Grep(ctx context.Context, args *GrepArgs) (*GrepResult, error) {
    workDir, err := p.getWorkDir()
    if err != nil {
        return nil, err
    }

    // Compile regex
    flags := ""
    if args.IgnoreCase {
        flags = "(?i)"
    }
    re, err := regexp.Compile(flags + args.Pattern)
    if err != nil {
        return nil, fmt.Errorf("invalid pattern: %w", err)
    }

    // Determine search path
    searchPath := workDir
    if args.Path != "" {
        searchPath, err = validatePath(workDir, args.Path)
        if err != nil {
            return nil, err
        }
    }

    // Walk directory
    maxResults := args.MaxResults
    if maxResults == 0 {
        maxResults = 100
    }

    matches := []GrepMatch{}
    err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil // Skip errors
        }

        // Skip directories and non-text files
        if info.IsDir() || !isTextFile(path) {
            return nil
        }

        // Read file
        content, err := os.ReadFile(path)
        if err != nil {
            return nil // Skip unreadable files
        }

        // Search lines
        lines := bytes.Split(content, []byte("\n"))
        for i, line := range lines {
            if re.Match(line) {
                relPath, _ := filepath.Rel(workDir, path)
                matches = append(matches, GrepMatch{
                    File:    relPath,
                    Line:    i + 1,
                    Content: string(line),
                })

                if len(matches) >= maxResults {
                    return filepath.SkipAll
                }
            }
        }

        return nil
    })

    if err != nil && err != filepath.SkipAll {
        return nil, fmt.Errorf("grep failed: %w", err)
    }

    return &GrepResult{
        Matches: matches,
        Total:   len(matches),
    }, nil
}

func isTextFile(path string) bool {
    // Simple heuristic: check extension
    ext := strings.ToLower(filepath.Ext(path))
    textExts := []string{
        ".txt", ".md", ".go", ".ts", ".js", ".tsx", ".jsx",
        ".py", ".rb", ".java", ".c", ".cpp", ".h", ".hpp",
        ".json", ".yaml", ".yml", ".toml", ".xml", ".html",
        ".css", ".scss", ".sql", ".sh", ".bash",
    }

    for _, textExt := range textExts {
        if ext == textExt {
            return true
        }
    }

    return false
}
```

### 6.5 Implement Glob

**pkg/providers/workspace/glob.go:**

```go
type GlobArgs struct {
    Pattern string `json:"pattern"`
}

type GlobResult struct {
    Files []string `json:"files"`
    Total int      `json:"total"`
}

func (p *Provider) Glob(ctx context.Context, args *GlobArgs) (*GlobResult, error) {
    workDir, err := p.getWorkDir()
    if err != nil {
        return nil, err
    }

    // Resolve pattern relative to work dir
    pattern := filepath.Join(workDir, args.Pattern)

    // Use filepath.Glob
    matches, err := filepath.Glob(pattern)
    if err != nil {
        return nil, fmt.Errorf("glob failed: %w", err)
    }

    // Convert to relative paths
    relMatches := make([]string, len(matches))
    for i, match := range matches {
        relPath, err := filepath.Rel(workDir, match)
        if err != nil {
            relPath = match
        }
        relMatches[i] = relPath
    }

    // Sort results
    sort.Strings(relMatches)

    return &GlobResult{
        Files: relMatches,
        Total: len(relMatches),
    }, nil
}
```

### 6.6 Register Tools

**pkg/providers/workspace/provider.go (continued):**

```go
func (p *Provider) registerTools() error {
    tools := []struct {
        tool mcp.Tool
        fn   mcp.ToolFunc
    }{
        {
            tool: mcp.Tool{
                Name:        "read_file",
                Description: "Read a file from the workspace",
                InputSchema: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "file_path": map[string]any{
                            "type": "string",
                            "description": "Path to file relative to workspace",
                        },
                        "offset": map[string]any{
                            "type": "integer",
                            "description": "Line number to start reading (1-indexed)",
                        },
                        "limit": map[string]any{
                            "type": "integer",
                            "description": "Number of lines to read",
                        },
                    },
                    "required": []string{"file_path"},
                },
            },
            fn: p.handleReadFile,
        },
        {
            tool: mcp.Tool{
                Name:        "write_file",
                Description: "Write a file to the workspace",
                InputSchema: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "file_path": map[string]any{
                            "type": "string",
                            "description": "Path to file relative to workspace",
                        },
                        "content": map[string]any{
                            "type": "string",
                            "description": "Content to write",
                        },
                    },
                    "required": []string{"file_path", "content"},
                },
            },
            fn: p.handleWriteFile,
        },
        // Add edit_file, bash, grep, glob tools...
    }

    for _, t := range tools {
        if err := p.registry.Register(t.tool, t.fn); err != nil {
            return err
        }
    }

    return nil
}

func (p *Provider) handleReadFile(ctx context.Context, params json.RawMessage) (*mcp.ToolResult, error) {
    var args ReadFileArgs
    if err := json.Unmarshal(params, &args); err != nil {
        return nil, err
    }

    content, err := p.ReadFile(ctx, &args)
    if err != nil {
        return nil, err
    }

    return &mcp.ToolResult{
        Content: []mcp.Content{{Type: "text", Text: content}},
    }, nil
}

// ... other handlers
```

### 6.7 Add Path Validation

**pkg/providers/workspace/pathutil.go:**

```go
func validatePath(baseDir, userPath string) (string, error) {
    // Clean path
    cleaned := filepath.Clean(userPath)

    // Check for absolute path attempts
    if filepath.IsAbs(cleaned) {
        return "", errors.New("absolute paths not allowed")
    }

    // Join with base
    fullPath := filepath.Join(baseDir, cleaned)

    // Resolve symlinks
    resolved, err := filepath.EvalSymlinks(fullPath)
    if err != nil {
        // File may not exist yet, check parent
        parent := filepath.Dir(fullPath)
        if parentResolved, err := filepath.EvalSymlinks(parent); err == nil {
            // Parent exists, construct full path
            resolved = filepath.Join(parentResolved, filepath.Base(fullPath))
        } else {
            return "", fmt.Errorf("parent directory does not exist: %s", parent)
        }
    }

    // Ensure within base directory
    baseResolved, err := filepath.EvalSymlinks(baseDir)
    if err != nil {
        return "", fmt.Errorf("failed to resolve base directory: %w", err)
    }

    if !strings.HasPrefix(resolved, baseResolved) {
        return "", errors.New("path outside workspace directory")
    }

    return resolved, nil
}
```

### 6.8 Write Tests

**pkg/providers/workspace/files_test.go:**

```go
func TestProvider_ReadFile(t *testing.T)
func TestProvider_WriteFile(t *testing.T)
func TestProvider_EditFile(t *testing.T)
func TestProvider_ReadFileWithRange(t *testing.T)
func TestProvider_EditFileNonUnique(t *testing.T)
```

**pkg/providers/workspace/bash_test.go:**

```go
func TestProvider_Bash(t *testing.T)
func TestProvider_BashTimeout(t *testing.T)
func TestProvider_BashNonZeroExit(t *testing.T)
```

**pkg/providers/workspace/grep_test.go:**

```go
func TestProvider_Grep(t *testing.T)
func TestProvider_GrepCaseInsensitive(t *testing.T)
func TestProvider_GrepMaxResults(t *testing.T)
```

**pkg/providers/workspace/security_test.go:**

```go
func TestValidatePath_TraversalAttempt(t *testing.T)
func TestValidatePath_SymlinkEscape(t *testing.T)
func TestValidatePath_AbsolutePath(t *testing.T)
```

## Acceptance Criteria

- [ ] File read/write/edit operations work
- [ ] Path validation prevents directory traversal
- [ ] Bash execution captures output correctly
- [ ] Grep searches files with regex
- [ ] Glob matches file patterns
- [ ] All operations scoped to workspace directory
- [ ] All unit tests pass
- [ ] Security tests prevent path escape

## Testing Commands

```bash
# Unit tests
go test ./pkg/providers/workspace/...

# Security tests
go test -run Security ./pkg/providers/workspace/...

# Integration test
go test ./test/integration/workspace_test.go
```

## Next Steps

Proceed to Step 7: Integration and Polish once all acceptance criteria are met.
