# Phase 1: Dagger Sandbox Implementation

## Objective

Implement a fully functional Dagger-based containerized sandbox for secure, isolated validation of data applications. This replaces the current stub implementation with a production-ready container orchestration system.

## Priority

**🔴 CRITICAL** - This is the most important gap in feature parity with the Rust implementation.

## Context

The Rust implementation at `edda_sandbox/src/dagger.rs` provides the reference implementation. The Go version currently has only a stub at `pkg/sandbox/dagger/dagger.go` that returns `ErrNotImplemented` for all operations.

**Impact of Current Gap:**
- Validation runs in local environment instead of isolated containers
- Security concerns: no process isolation
- Inconsistent validation environments across machines
- Cannot guarantee clean state between validations

## Prerequisites

- Dagger SDK for Go: `dagger.io/dagger`
- Docker daemon must be running
- Study Rust implementation: `/Users/fabian.jakobs/Workspaces/agent/edda/edda_sandbox/src/dagger.rs`
- Review Dagger Go SDK documentation

## Implementation Steps

### Step 1: Dagger SDK Integration

#### 1.1 Add Dependencies
```bash
cd /Users/fabian.jakobs/Workspaces/app-mcp
go get dagger.io/dagger@latest
```

#### 1.2 Study Reference Implementation
Read and understand these files:
- Rust: `/Users/fabian.jakobs/Workspaces/agent/edda/edda_sandbox/src/dagger.rs`
- Current stub: `pkg/sandbox/dagger/dagger.go`
- Interface: `pkg/sandbox/sandbox.go`

#### 1.3 Review API Differences
Compare Dagger Rust SDK vs Go SDK:
- Container creation patterns
- File operations API
- Command execution
- Directory mounting
- Error handling

### Step 2: Core Data Structures

#### 2.1 DaggerSandbox Struct

Replace the stub in `pkg/sandbox/dagger/dagger.go`:

```go
package dagger

import (
    "context"
    "fmt"
    "path/filepath"
    "sync"

    "dagger.io/dagger"
    "go-mcp/pkg/sandbox"
)

// DaggerSandbox implements the Sandbox interface using Dagger
type DaggerSandbox struct {
    client    *dagger.Client
    container *dagger.Container
    workdir   string
    baseDir   string
    mu        sync.RWMutex

    // Configuration
    image         string
    executeTimeout int  // seconds
}

// Config holds Dagger sandbox configuration
type Config struct {
    Image          string // Docker image (default: node:20-alpine3.22)
    ExecuteTimeout int    // Execution timeout in seconds (default: 600)
    BaseDir        string // Base directory for operations
}
```

#### 2.2 Constructor

```go
// NewDaggerSandbox creates a new Dagger-based sandbox
// Reference: edda_sandbox/src/dagger.rs:45-76
func NewDaggerSandbox(ctx context.Context, cfg Config) (*DaggerSandbox, error) {
    if cfg.Image == "" {
        cfg.Image = "node:20-alpine3.22"
    }
    if cfg.ExecuteTimeout == 0 {
        cfg.ExecuteTimeout = 600
    }

    client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Dagger: %w", err)
    }

    container := client.Container().
        From(cfg.Image).
        WithWorkdir("/workspace")

    return &DaggerSandbox{
        client:         client,
        container:      container,
        workdir:        "/workspace",
        baseDir:        cfg.BaseDir,
        image:          cfg.Image,
        executeTimeout: cfg.ExecuteTimeout,
    }, nil
}
```

### Step 3: Core Operations

#### 3.1 Exec - Execute Commands

```go
// Exec executes a command in the sandbox
// Reference: edda_sandbox/src/dagger.rs:78-105
func (d *DaggerSandbox) Exec(ctx context.Context, command string) (*sandbox.ExecResult, error) {
    d.mu.RLock()
    container := d.container.WithExec([]string{"sh", "-c", command})
    d.mu.RUnlock()

    // Execute and get output
    stdout, err := container.Stdout(ctx)
    if err != nil {
        stderr, _ := container.Stderr(ctx)
        exitCode, _ := container.ExitCode(ctx)

        return &sandbox.ExecResult{
            Stdout:   stdout,
            Stderr:   stderr,
            ExitCode: exitCode,
        }, fmt.Errorf("command failed: %w", err)
    }

    stderr, _ := container.Stderr(ctx)
    exitCode, _ := container.ExitCode(ctx)

    return &sandbox.ExecResult{
        Stdout:   stdout,
        Stderr:   stderr,
        ExitCode: exitCode,
    }, nil
}
```

#### 3.2 WriteFile - Single File Write

```go
// WriteFile writes content to a file in the sandbox
// Reference: edda_sandbox/src/dagger.rs:107-123
func (d *DaggerSandbox) WriteFile(ctx context.Context, path string, content string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    fullPath := filepath.Join(d.workdir, path)

    // Create directory structure if needed
    dir := filepath.Dir(fullPath)
    d.container = d.container.WithExec([]string{"mkdir", "-p", dir})

    // Write file via WithNewFile
    d.container = d.container.WithNewFile(fullPath, dagger.ContainerWithNewFileOpts{
        Contents:    content,
        Permissions: 0644,
    })

    // Force evaluation
    _, err := d.container.Sync(ctx)
    return err
}
```

#### 3.3 WriteFiles - Bulk File Operations

**CRITICAL OPTIMIZATION**: Use temporary directory pattern to prevent deep query chains.

```go
// WriteFiles writes multiple files atomically
// Reference: edda_sandbox/src/dagger.rs:125-167
func (d *DaggerSandbox) WriteFiles(ctx context.Context, files map[string]string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    // Create temporary directory with all files
    tmpDir := d.client.Directory()

    for path, content := range files {
        tmpDir = tmpDir.WithNewFile(path, content)
    }

    // Mount the directory into the container at workdir
    d.container = d.container.WithDirectory(d.workdir, tmpDir)

    // Force evaluation to ensure writes complete
    _, err := d.container.Sync(ctx)
    return err
}
```

#### 3.4 ReadFile - Read File Contents

```go
// ReadFile reads a file from the sandbox
// Reference: edda_sandbox/src/dagger.rs:169-181
func (d *DaggerSandbox) ReadFile(ctx context.Context, path string) (string, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    fullPath := filepath.Join(d.workdir, path)

    file := d.container.File(fullPath)
    contents, err := file.Contents(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to read file %s: %w", path, err)
    }

    return contents, nil
}
```

#### 3.5 DeleteFile - Remove File

```go
// DeleteFile removes a file from the sandbox
// Reference: edda_sandbox/src/dagger.rs:183-195
func (d *DaggerSandbox) DeleteFile(ctx context.Context, path string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    fullPath := filepath.Join(d.workdir, path)
    d.container = d.container.WithExec([]string{"rm", "-f", fullPath})

    _, err := d.container.Sync(ctx)
    return err
}
```

#### 3.6 ListDirectory - List Directory Contents

```go
// ListDirectory lists entries in a directory
// Reference: edda_sandbox/src/dagger.rs:197-214
func (d *DaggerSandbox) ListDirectory(ctx context.Context, path string) ([]sandbox.DirEntry, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    fullPath := filepath.Join(d.workdir, path)

    dir := d.container.Directory(fullPath)
    entries, err := dir.Entries(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to list directory %s: %w", path, err)
    }

    result := make([]sandbox.DirEntry, len(entries))
    for i, name := range entries {
        result[i] = sandbox.DirEntry{
            Name:  name,
            IsDir: false, // Dagger doesn't distinguish, may need stat
        }
    }

    return result, nil
}
```

#### 3.7 SetWorkdir - Change Working Directory

```go
// SetWorkdir changes the working directory
// Reference: edda_sandbox/src/dagger.rs:216-227
func (d *DaggerSandbox) SetWorkdir(ctx context.Context, path string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    d.workdir = path
    d.container = d.container.WithWorkdir(path)

    return nil
}
```

### Step 4: Advanced Operations

#### 4.1 ExportDirectory - Export to Host

```go
// ExportDirectory exports a container directory to the host
// Reference: edda_sandbox/src/dagger.rs:229-246
func (d *DaggerSandbox) ExportDirectory(ctx context.Context, containerPath, hostPath string) error {
    d.mu.RLock()
    dir := d.container.Directory(containerPath)
    d.mu.RUnlock()

    // Export to host filesystem
    _, err := dir.Export(ctx, hostPath)
    if err != nil {
        return fmt.Errorf("failed to export directory: %w", err)
    }

    return nil
}
```

#### 4.2 RefreshFromHost - Import from Host

```go
// RefreshFromHost syncs host directory into container
// Reference: edda_sandbox/src/dagger.rs:248-265
func (d *DaggerSandbox) RefreshFromHost(ctx context.Context, hostPath, containerPath string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    // Load host directory
    hostDir := d.client.Host().Directory(hostPath)

    // Mount into container
    d.container = d.container.WithDirectory(containerPath, hostDir)

    _, err := d.container.Sync(ctx)
    return err
}
```

#### 4.3 Fork - Clone Sandbox State

```go
// Fork creates a copy of the sandbox with the same state
// Useful for parallel operations
func (d *DaggerSandbox) Fork() sandbox.Sandbox {
    d.mu.RLock()
    defer d.mu.RUnlock()

    return &DaggerSandbox{
        client:         d.client,
        container:      d.container, // Dagger containers are immutable
        workdir:        d.workdir,
        baseDir:        d.baseDir,
        image:          d.image,
        executeTimeout: d.executeTimeout,
    }
}
```

#### 4.4 Close - Cleanup Resources

```go
// Close releases Dagger client resources
func (d *DaggerSandbox) Close() error {
    if d.client != nil {
        return d.client.Close()
    }
    return nil
}
```

### Step 5: Environment Variable Propagation

Add support for passing environment variables to the container:

```go
// WithEnv adds environment variables to the container
func (d *DaggerSandbox) WithEnv(key, value string) {
    d.mu.Lock()
    defer d.mu.Unlock()

    d.container = d.container.WithEnvVariable(key, value)
}

// WithEnvs adds multiple environment variables
func (d *DaggerSandbox) WithEnvs(envs map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()

    for key, value := range envs {
        d.container = d.container.WithEnvVariable(key, value)
    }
}
```

### Step 6: Integration with IO Provider

Update `pkg/providers/io/validate.go` to use Dagger sandbox:

```go
func (p *Provider) validateDataApp(ctx context.Context, args ValidateDataAppInput) (*ValidateDataAppResult, error) {
    // Create Dagger sandbox
    sandbox, err := dagger.NewDaggerSandbox(ctx, dagger.Config{
        Image:   "node:20-alpine3.22",
        BaseDir: args.WorkDir,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create sandbox: %w", err)
    }
    defer sandbox.Close()

    // Propagate DATABRICKS_* environment variables
    for _, key := range []string{"DATABRICKS_HOST", "DATABRICKS_TOKEN", "DATABRICKS_WAREHOUSE_ID"} {
        if value := os.Getenv(key); value != "" {
            sandbox.WithEnv(key, value)
        }
    }

    // Copy project files to sandbox
    if err := sandbox.RefreshFromHost(ctx, args.WorkDir, "/workspace"); err != nil {
        return nil, fmt.Errorf("failed to copy files to sandbox: %w", err)
    }

    // Run validation strategy
    // ... rest of validation logic
}
```

## Testing

### Unit Tests (`pkg/sandbox/dagger/dagger_test.go`)

```go
package dagger

import (
    "context"
    "testing"
)

func TestDaggerSandbox_Exec(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    result, err := sandbox.Exec(ctx, "echo 'hello world'")
    if err != nil {
        t.Fatalf("exec failed: %v", err)
    }

    if result.ExitCode != 0 {
        t.Errorf("expected exit code 0, got %d", result.ExitCode)
    }

    // Note: stdout might have newline
    if !strings.Contains(result.Stdout, "hello world") {
        t.Errorf("expected 'hello world' in stdout, got: %s", result.Stdout)
    }
}

func TestDaggerSandbox_WriteReadFile(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    content := "test content"
    if err := sandbox.WriteFile(ctx, "test.txt", content); err != nil {
        t.Fatalf("write failed: %v", err)
    }

    read, err := sandbox.ReadFile(ctx, "test.txt")
    if err != nil {
        t.Fatalf("read failed: %v", err)
    }

    if read != content {
        t.Errorf("expected %q, got %q", content, read)
    }
}

func TestDaggerSandbox_WriteFiles_BulkOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    files := map[string]string{
        "file1.txt": "content1",
        "file2.txt": "content2",
        "dir/file3.txt": "content3",
    }

    if err := sandbox.WriteFiles(ctx, files); err != nil {
        t.Fatalf("write files failed: %v", err)
    }

    // Verify all files exist
    for path, expected := range files {
        content, err := sandbox.ReadFile(ctx, path)
        if err != nil {
            t.Errorf("failed to read %s: %v", path, err)
        }
        if content != expected {
            t.Errorf("%s: expected %q, got %q", path, expected, content)
        }
    }
}

func TestDaggerSandbox_RefreshFromHost(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    // Create temporary directory with test files
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.txt")
    if err := os.WriteFile(testFile, []byte("host content"), 0644); err != nil {
        t.Fatalf("failed to create test file: %v", err)
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    // Sync host dir to container
    if err := sandbox.RefreshFromHost(ctx, tmpDir, "/workspace"); err != nil {
        t.Fatalf("refresh failed: %v", err)
    }

    // Verify file exists in container
    content, err := sandbox.ReadFile(ctx, "test.txt")
    if err != nil {
        t.Fatalf("read failed: %v", err)
    }

    if content != "host content" {
        t.Errorf("expected 'host content', got %q", content)
    }
}
```

### Integration Tests (`test/integration/dagger_validation_test.go`)

Test full validation workflow with Dagger:

```go
func TestDaggerValidation_TRPCProject(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create test project
    // Scaffold tRPC template
    // Validate using Dagger sandbox
    // Verify validation passes
}
```

### Performance Benchmarks

```go
func BenchmarkDagger_vs_Local(b *testing.B) {
    // Compare Dagger vs Local sandbox performance
    // Measure: file writes, command execution, bulk operations
}
```

## Configuration Updates

Update `pkg/config/config.go`:

```go
type ValidationConfig struct {
    Command     string `json:"command,omitempty"`
    DockerImage string `json:"docker_image,omitempty"` // Default: node:20-alpine3.22
    UseDagger   bool   `json:"use_dagger"`             // Default: true
    Timeout     int    `json:"timeout,omitempty"`      // Default: 600 seconds
}
```

## Documentation Updates

### CLAUDE.md Updates

Add to the Sandbox section:

```markdown
### Dagger Implementation (pkg/sandbox/dagger)

**Fully Implemented** containerized sandbox:

**Container Operations:**
- `exec()` - Execute commands via `sh -c`
- `write_file()` - Single file write
- `write_files()` - Bulk file write (optimized with temp dir pattern)
- `read_file()` - Read file contents
- `delete_file()` - Remove file
- `list_directory()` - List directory entries
- `set_workdir()` - Set working directory
- `export_directory()` - Export container dir to host
- `refresh_from_host()` - Sync host dir to container
- `fork()` - Clone sandbox state
- `close()` - Cleanup resources

**Security Features:**
- Process isolation via containers
- Configurable Docker image
- Execution timeout support (default: 600s)
- Environment variable propagation
```

## Verification Checklist

After implementation:
- [ ] All Dagger operations implemented
- [ ] Unit tests pass (`go test ./pkg/sandbox/dagger/`)
- [ ] Integration tests pass with Docker running
- [ ] Validation uses Dagger by default
- [ ] Falls back to local if Docker unavailable
- [ ] Environment variables propagated correctly
- [ ] Performance benchmarks show acceptable overhead (<2x vs local)
- [ ] Documentation updated (CLAUDE.md, code comments)
- [ ] No compilation errors or warnings
- [ ] Race detector passes (`go test -race`)

## Success Criteria

- [ ] All sandbox interface methods fully implemented
- [ ] Tests pass with real Dagger containers
- [ ] Validation workflow uses Dagger successfully
- [ ] Performance overhead <100% vs local sandbox
- [ ] Graceful fallback when Docker unavailable
- [ ] Environment variable propagation working
- [ ] Code follows Go idioms
- [ ] Documentation complete
- [ ] Behavioral parity with Rust implementation

## Rollout Strategy

1. **Development**: Implement behind feature flag `use_dagger: false`
2. **Testing**: Enable for internal testing
3. **Beta**: Enable by default with local fallback
4. **Production**: Dagger-only (fail if Docker unavailable)

## Known Issues / Gotchas

1. **Dagger Query Chains**: Avoid deep chains, use `Sync()` to force evaluation
2. **File Permissions**: Default to 0644, may need configuration
3. **Network Access**: Containers may need network for npm install
4. **Volume Mounts**: Understand Dagger's immutability model
5. **Error Messages**: Container errors may be cryptic, add context

## Timeline

- **Week 1**: Steps 1-3 (SDK integration, core operations)
- **Week 2**: Steps 4-5 (advanced operations, env vars)
- **Week 3**: Step 6 (IO provider integration), testing
- **Week 4**: Performance tuning, documentation, polish

## Dependencies

- Dagger SDK: `dagger.io/dagger@latest`
- Docker daemon running locally
- Go 1.21+ for generics support

## Next Phase

After completing Dagger sandbox, proceed to:
- **Phase 2**: Trajectory Tracking
