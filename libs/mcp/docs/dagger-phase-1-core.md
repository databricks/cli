# Dagger Phase 1: SDK Integration & Core Operations

**Issue**: parity-44
**Parent**: parity-2 (Phase 1: Dagger Sandbox Implementation)
**Estimated Time**: 1 week

## Objective

Implement Dagger SDK integration, core data structures, and basic file/command operations. This establishes the foundation for containerized validation.

## Prerequisites

- Dagger SDK for Go: `dagger.io/dagger`
- Docker daemon must be running
- Study Rust reference: `../agent/edda/edda_sandbox/src/dagger.rs`
- Review Dagger Go SDK documentation
- dagger support must be enabled explicitly through a configuration option. By default use the local sandbox

## Implementation Steps

### Step 1: Dagger SDK Integration

#### 1.1 Add Dependencies
```bash
cd /Users/fabian.jakobs/Workspaces/app-mcp
go get dagger.io/dagger@latest
```

#### 1.2 Study Reference Implementation
Read and understand:
- Rust: `../agent/edda/edda_sandbox/src/dagger.rs` (lines 1-265)
- Current stub: `pkg/sandbox/dagger/dagger.go`
- Interface: `pkg/sandbox/sandbox.go`

#### 1.3 Review API Differences
Compare Dagger Rust SDK vs Go SDK:
- Container creation patterns
- File operations API
- Command execution patterns
- Directory mounting
- Error handling patterns

### Step 2: Core Data Structures

#### 2.1 DaggerSandbox Struct

Replace stub in `pkg/sandbox/dagger/dagger.go`:

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

type DaggerSandbox struct {
    client    *dagger.Client
    container *dagger.Container
    workdir   string
    baseDir   string
    mu        sync.RWMutex

    image         string
    executeTimeout int  // seconds
}

type Config struct {
    Image          string // Docker image (default: node:20-alpine3.22)
    ExecuteTimeout int    // Execution timeout in seconds (default: 600)
    BaseDir        string // Base directory for operations
}
```

#### 2.2 Constructor

**Reference**: `edda_sandbox/src/dagger.rs:45-76`

```go
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

**Reference**: `edda_sandbox/src/dagger.rs:78-105`

```go
func (d *DaggerSandbox) Exec(ctx context.Context, command string) (*sandbox.ExecResult, error) {
    d.mu.RLock()
    container := d.container.WithExec([]string{"sh", "-c", command})
    d.mu.RUnlock()

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

**Reference**: `edda_sandbox/src/dagger.rs:107-123`

```go
func (d *DaggerSandbox) WriteFile(ctx context.Context, path string, content string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    fullPath := filepath.Join(d.workdir, path)

    dir := filepath.Dir(fullPath)
    d.container = d.container.WithExec([]string{"mkdir", "-p", dir})

    d.container = d.container.WithNewFile(fullPath, dagger.ContainerWithNewFileOpts{
        Contents:    content,
        Permissions: 0644,
    })

    _, err := d.container.Sync(ctx)
    return err
}
```

#### 3.3 WriteFiles - Bulk File Operations

**Reference**: `edda_sandbox/src/dagger.rs:125-167`

**CRITICAL OPTIMIZATION**: Use temporary directory pattern to prevent deep query chains.

```go
func (d *DaggerSandbox) WriteFiles(ctx context.Context, files map[string]string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    tmpDir := d.client.Directory()

    for path, content := range files {
        tmpDir = tmpDir.WithNewFile(path, content)
    }

    d.container = d.container.WithDirectory(d.workdir, tmpDir)

    _, err := d.container.Sync(ctx)
    return err
}
```

#### 3.4 ReadFile - Read File Contents

**Reference**: `edda_sandbox/src/dagger.rs:169-181`

```go
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

**Reference**: `edda_sandbox/src/dagger.rs:183-195`

```go
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

**Reference**: `edda_sandbox/src/dagger.rs:197-214`

```go
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
            IsDir: false,
        }
    }

    return result, nil
}
```

#### 3.7 SetWorkdir - Change Working Directory

**Reference**: `edda_sandbox/src/dagger.rs:216-227`

```go
func (d *DaggerSandbox) SetWorkdir(ctx context.Context, path string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    d.workdir = path
    d.container = d.container.WithWorkdir(path)

    return nil
}
```

## Testing

Create basic unit tests in `pkg/sandbox/dagger/dagger_test.go`:

```go
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
```

## Verification Checklist

- [ ] Dagger SDK dependency added
- [ ] Reference implementation studied and understood
- [ ] DaggerSandbox struct and Config defined
- [ ] NewDaggerSandbox constructor implemented
- [ ] Exec operation working
- [ ] WriteFile operation working
- [ ] WriteFiles bulk operation working
- [ ] ReadFile operation working
- [ ] DeleteFile operation working
- [ ] ListDirectory operation working
- [ ] SetWorkdir operation working
- [ ] Basic unit tests pass
- [ ] Thread-safe with proper mutex usage
- [ ] No compilation errors
- [ ] dagger is off by default and can be enabled through command line, config file or environment variable

## Success Criteria

- [ ] All core operations implemented and working
- [ ] Unit tests pass with Docker running
- [ ] Code follows Go idioms (proper error handling, thread safety)
- [ ] Behavioral parity with Rust implementation (lines 45-227)
- [ ] Documentation comments on all public functions

## Next Steps

After completing this phase, proceed to:
- **parity-45**: Advanced Operations & Environment Support
