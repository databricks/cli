# Step 2: Sandbox Abstraction

## Overview
Implement a clean abstraction for executing commands and managing files in an isolated environment. Start with local filesystem implementation, with hooks for future Dagger backend.

## Tasks

### 2.1 Define Sandbox Interface

**pkg/sandbox/sandbox.go:**

```go
type ExecResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
}

type Sandbox interface {
    // Execute a command in the sandbox
    Exec(ctx context.Context, command string) (*ExecResult, error)

    // File operations
    WriteFile(ctx context.Context, path, content string) error
    WriteFiles(ctx context.Context, files map[string]string) error
    ReadFile(ctx context.Context, path string) (string, error)
    DeleteFile(ctx context.Context, path string) error

    // Directory operations
    ListDirectory(ctx context.Context, path string) ([]string, error)
    SetWorkdir(ctx context.Context, path string) error

    // Export/import operations
    ExportDirectory(ctx context.Context, containerPath, hostPath string) (string, error)
    RefreshFromHost(ctx context.Context, hostPath, containerPath string) error

    // Lifecycle
    Close() error
}
```

### 2.2 Implement Local Filesystem Sandbox

**pkg/sandbox/local/local.go:**

```go
type LocalSandbox struct {
    baseDir string        // Base directory for all operations
    workDir string        // Current working directory (relative to baseDir)
    mu      sync.RWMutex  // Protects directory state
}

func NewLocalSandbox(baseDir string) (*LocalSandbox, error)
```

Implementation details:
- All paths must be validated to stay within `baseDir`
- `Exec` uses `os/exec.Command` with working directory set
- File operations use standard `os` and `io/fs` packages
- Implement proper path normalization and validation
- Handle symlinks carefully to prevent escaping baseDir

Key methods:

**Exec:**
```go
func (s *LocalSandbox) Exec(ctx context.Context, command string) (*ExecResult, error) {
    // Parse command into parts
    // Use exec.CommandContext for cancellation
    // Capture stdout/stderr with bytes.Buffer
    // Set working directory
    // Return ExecResult with exit code
}
```

**WriteFile:**
```go
func (s *LocalSandbox) WriteFile(ctx context.Context, path, content string) error {
    // Validate path within baseDir
    // Create parent directories if needed
    // Write file atomically (write to temp, then rename)
    // Set appropriate permissions
}
```

**ReadFile:**
```go
func (s *LocalSandbox) ReadFile(ctx context.Context, path string) (string, error) {
    // Validate path
    // Read file with io.ReadFile
    // Return content as string
}
```

**ListDirectory:**
```go
func (s *LocalSandbox) ListDirectory(ctx context.Context, path string) ([]string, error) {
    // Validate path
    // Use os.ReadDir
    // Return sorted list of names
}
```

### 2.3 Implement Path Validation

**pkg/sandbox/local/pathutil.go:**

```go
// ValidatePath ensures path is within baseDir and returns absolute path
func ValidatePath(baseDir, userPath string) (string, error) {
    base, err := filepath.Abs(baseDir)
    if err != nil {
        return "", err
    }

    target := filepath.Join(base, userPath)
    cleaned := filepath.Clean(target)

    // Resolve symlinks
    resolved, err := filepath.EvalSymlinks(cleaned)
    if err != nil {
        // If file doesn't exist, check parent
        parent := filepath.Dir(cleaned)
        if _, err := os.Stat(parent); os.IsNotExist(err) {
            return "", fmt.Errorf("parent directory does not exist")
        }
        resolved = cleaned
    }

    // Check if within base
    if !strings.HasPrefix(resolved, base) {
        return "", fmt.Errorf("path outside base directory")
    }

    return resolved, nil
}
```

### 2.4 Create Dagger Backend Stub

**pkg/sandbox/dagger/dagger.go:**

```go
type DaggerSandbox struct {
    // TODO: Dagger client fields
}

func NewDaggerSandbox() (*DaggerSandbox, error) {
    return nil, errors.New("Dagger backend not yet implemented")
}

// Implement Sandbox interface with stub methods
func (d *DaggerSandbox) Exec(ctx context.Context, command string) (*ExecResult, error) {
    return nil, errors.New("not implemented")
}

// ... other methods
```

### 2.5 Add Sandbox Factory

**pkg/sandbox/factory.go:**

```go
type Type string

const (
    TypeLocal  Type = "local"
    TypeDagger Type = "dagger"
)

func New(t Type, opts ...Option) (Sandbox, error) {
    switch t {
    case TypeLocal:
        return local.NewLocalSandbox(/* options */)
    case TypeDagger:
        return nil, errors.New("Dagger backend not yet available")
    default:
        return nil, fmt.Errorf("unknown sandbox type: %s", t)
    }
}

type Option func(*config)

func WithBaseDir(dir string) Option
func WithTimeout(d time.Duration) Option
```

### 2.6 Write Comprehensive Tests

**pkg/sandbox/local/local_test.go:**

Test cases:
```go
func TestLocalSandbox_Exec(t *testing.T)
func TestLocalSandbox_WriteFile(t *testing.T)
func TestLocalSandbox_ReadFile(t *testing.T)
func TestLocalSandbox_ListDirectory(t *testing.T)
func TestLocalSandbox_SetWorkdir(t *testing.T)
func TestLocalSandbox_PathTraversal(t *testing.T)  // Security test
func TestLocalSandbox_SymlinkEscape(t *testing.T)  // Security test
func TestLocalSandbox_Concurrent(t *testing.T)     // Concurrency test
```

Example test structure:
```go
func TestLocalSandbox_WriteFile(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        content string
        wantErr bool
        errMsg  string
    }{
        {
            name:    "simple file",
            path:    "test.txt",
            content: "hello world",
            wantErr: false,
        },
        {
            name:    "nested path",
            path:    "subdir/test.txt",
            content: "nested",
            wantErr: false,
        },
        {
            name:    "path traversal attempt",
            path:    "../outside.txt",
            content: "malicious",
            wantErr: true,
            errMsg:  "outside base directory",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create temp base directory
            baseDir := t.TempDir()
            sb, err := NewLocalSandbox(baseDir)
            require.NoError(t, err)

            // Run test
            err = sb.WriteFile(context.Background(), tt.path, tt.content)
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
                // Verify file was written
                content, err := sb.ReadFile(context.Background(), tt.path)
                require.NoError(t, err)
                assert.Equal(t, tt.content, content)
            }
        })
    }
}
```

### 2.7 Add Integration Tests

**test/integration/sandbox_test.go:**

Test realistic workflows:
```go
func TestSandbox_NPMWorkflow(t *testing.T) {
    // Skip if docker not available
    if !dockerAvailable() {
        t.Skip("Docker not available")
    }

    baseDir := t.TempDir()
    sb, err := sandbox.New(sandbox.TypeLocal,
        sandbox.WithBaseDir(baseDir))
    require.NoError(t, err)
    defer sb.Close()

    // Create package.json
    err = sb.WriteFile(context.Background(), "package.json", `{
        "name": "test",
        "scripts": {"test": "echo success"}
    }`)
    require.NoError(t, err)

    // Run npm install (if npm available)
    result, err := sb.Exec(context.Background(), "npm test")
    require.NoError(t, err)
    assert.Equal(t, 0, result.ExitCode)
    assert.Contains(t, result.Stdout, "success")
}
```

### 2.8 Add Benchmarks

**pkg/sandbox/local/local_bench_test.go:**

```go
func BenchmarkLocalSandbox_WriteFile(b *testing.B)
func BenchmarkLocalSandbox_ReadFile(b *testing.B)
func BenchmarkLocalSandbox_Exec(b *testing.B)
```

## Acceptance Criteria

- [ ] Sandbox interface defined with complete API
- [ ] Local sandbox implementation working
- [ ] Path validation prevents directory traversal
- [ ] All file operations work correctly
- [ ] Command execution captures stdout/stderr
- [ ] Working directory can be changed
- [ ] All unit tests pass (including security tests)
- [ ] Integration tests pass
- [ ] Test coverage >80% for sandbox package
- [ ] Dagger stub created for future implementation

## Testing Commands

```bash
# Run sandbox tests
go test ./pkg/sandbox/...

# Run with race detector
go test -race ./pkg/sandbox/...

# Run benchmarks
go test -bench=. ./pkg/sandbox/local/

# Run integration tests
go test ./test/integration/...

# Coverage report
go test -coverprofile=coverage.out ./pkg/sandbox/...
go tool cover -html=coverage.out
```

## Security Considerations

- All paths MUST be validated before use
- Symlinks must be resolved and checked
- Never execute user input directly (sanitize commands)
- File permissions should be restrictive by default
- Consider resource limits (file size, execution time)
- Test for common attack vectors (path traversal, symlink escape)

## Next Steps

Proceed to Step 3: MCP Protocol Implementation once all acceptance criteria are met.
