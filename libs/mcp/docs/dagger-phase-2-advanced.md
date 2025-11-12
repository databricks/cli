# Dagger Phase 2: Advanced Operations & Environment Support

**Issue**: parity-45
**Parent**: parity-2 (Phase 1: Dagger Sandbox Implementation)
**Depends On**: parity-44 (Core Operations)
**Estimated Time**: 5 days

## Objective

Implement advanced sandbox operations (directory export/import, forking, cleanup) and environment variable propagation to support complex validation workflows.

## Prerequisites

- Phase 1 (parity-44) completed: Core operations working
- Docker daemon running
- Understanding of Dagger immutability model

## Implementation Steps

### Step 1: Advanced Directory Operations

#### 1.1 ExportDirectory - Export to Host

**Reference**: `edda_sandbox/src/dagger.rs:229-246`

Export container directory to host filesystem for artifact collection:

```go
func (d *DaggerSandbox) ExportDirectory(ctx context.Context, containerPath, hostPath string) error {
    d.mu.RLock()
    dir := d.container.Directory(containerPath)
    d.mu.RUnlock()

    _, err := dir.Export(ctx, hostPath)
    if err != nil {
        return fmt.Errorf("failed to export directory: %w", err)
    }

    return nil
}
```

**Use Case**: Export build artifacts, test results, logs from container to host.

#### 1.2 RefreshFromHost - Import from Host

**Reference**: `edda_sandbox/src/dagger.rs:248-265`

Sync host directory into container for project scaffolding:

```go
func (d *DaggerSandbox) RefreshFromHost(ctx context.Context, hostPath, containerPath string) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    hostDir := d.client.Host().Directory(hostPath)

    d.container = d.container.WithDirectory(containerPath, hostDir)

    _, err := d.container.Sync(ctx)
    return err
}
```

**Use Case**: Load scaffolded project files from host into container for validation.

### Step 2: Sandbox Lifecycle Management

#### 2.1 Fork - Clone Sandbox State

Create a copy of sandbox for parallel operations:

```go
func (d *DaggerSandbox) Fork() sandbox.Sandbox {
    d.mu.RLock()
    defer d.mu.RUnlock()

    return &DaggerSandbox{
        client:         d.client,
        container:      d.container,
        workdir:        d.workdir,
        baseDir:        d.baseDir,
        image:          d.image,
        executeTimeout: d.executeTimeout,
    }
}
```

**Use Case**: Run multiple validations in parallel with same base state.

**Note**: Dagger containers are immutable, so forking is lightweight (just pointer copy).

#### 2.2 Close - Cleanup Resources

Properly release Dagger client:

```go
func (d *DaggerSandbox) Close() error {
    if d.client != nil {
        return d.client.Close()
    }
    return nil
}
```

**Important**: Always call Close() via defer to prevent resource leaks.

### Step 3: Environment Variable Propagation

Add support for passing environment variables to containers:

#### 3.1 WithEnv - Single Variable

```go
func (d *DaggerSandbox) WithEnv(key, value string) {
    d.mu.Lock()
    defer d.mu.Unlock()

    d.container = d.container.WithEnvVariable(key, value)
}
```

#### 3.2 WithEnvs - Multiple Variables

```go
func (d *DaggerSandbox) WithEnvs(envs map[string]string) {
    d.mu.Lock()
    defer d.mu.Unlock()

    for key, value := range envs {
        d.container = d.container.WithEnvVariable(key, value)
    }
}
```

**Critical Environment Variables**:
- `DATABRICKS_HOST` - Workspace URL
- `DATABRICKS_TOKEN` - Authentication token
- `DATABRICKS_WAREHOUSE_ID` - SQL warehouse ID

**Security Note**: These methods should only be called internally by the provider, never exposed directly to MCP tools.

### Step 4: Timeout Handling

Implement execution timeout support:

```go
func (d *DaggerSandbox) ExecWithTimeout(ctx context.Context, command string) (*sandbox.ExecResult, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(d.executeTimeout)*time.Second)
    defer cancel()

    return d.Exec(timeoutCtx, command)
}
```

**Usage Pattern**:
```go
result, err := sandbox.ExecWithTimeout(ctx, "npm test")
if errors.Is(err, context.DeadlineExceeded) {
    return fmt.Errorf("validation timed out after %d seconds", sandbox.executeTimeout)
}
```

## Testing

### Unit Tests

Add to `pkg/sandbox/dagger/dagger_test.go`:

```go
func TestDaggerSandbox_RefreshFromHost(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

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

    if err := sandbox.RefreshFromHost(ctx, tmpDir, "/workspace"); err != nil {
        t.Fatalf("refresh failed: %v", err)
    }

    content, err := sandbox.ReadFile(ctx, "test.txt")
    if err != nil {
        t.Fatalf("read failed: %v", err)
    }

    if content != "host content" {
        t.Errorf("expected 'host content', got %q", content)
    }
}

func TestDaggerSandbox_ExportDirectory(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    if err := sandbox.WriteFile(ctx, "output.txt", "result data"); err != nil {
        t.Fatalf("write failed: %v", err)
    }

    tmpDir := t.TempDir()
    if err := sandbox.ExportDirectory(ctx, "/workspace", tmpDir); err != nil {
        t.Fatalf("export failed: %v", err)
    }

    content, err := os.ReadFile(filepath.Join(tmpDir, "output.txt"))
    if err != nil {
        t.Fatalf("read exported file failed: %v", err)
    }

    if string(content) != "result data" {
        t.Errorf("expected 'result data', got %q", string(content))
    }
}

func TestDaggerSandbox_WithEnv(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    sandbox.WithEnv("TEST_VAR", "test_value")

    result, err := sandbox.Exec(ctx, "echo $TEST_VAR")
    if err != nil {
        t.Fatalf("exec failed: %v", err)
    }

    if !strings.Contains(result.Stdout, "test_value") {
        t.Errorf("expected 'test_value' in stdout, got: %s", result.Stdout)
    }
}

func TestDaggerSandbox_Fork(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    original, err := NewDaggerSandbox(ctx, Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer original.Close()

    if err := original.WriteFile(ctx, "base.txt", "base content"); err != nil {
        t.Fatalf("write to original failed: %v", err)
    }

    forked := original.Fork()
    defer forked.Close()

    content, err := forked.ReadFile(ctx, "base.txt")
    if err != nil {
        t.Fatalf("read from fork failed: %v", err)
    }

    if content != "base content" {
        t.Errorf("expected 'base content', got %q", content)
    }
}

func TestDaggerSandbox_ExecWithTimeout(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping Dagger integration test")
    }

    ctx := context.Background()
    sandbox, err := NewDaggerSandbox(ctx, Config{ExecuteTimeout: 2})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sandbox.Close()

    _, err = sandbox.ExecWithTimeout(ctx, "sleep 5")
    if err == nil {
        t.Error("expected timeout error")
    }

    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("expected DeadlineExceeded, got: %v", err)
    }
}
```

## Verification Checklist

- [ ] ExportDirectory working (container → host)
- [ ] RefreshFromHost working (host → container)
- [ ] Fork creates independent copy
- [ ] Close properly releases resources
- [ ] WithEnv sets single environment variable
- [ ] WithEnvs sets multiple environment variables
- [ ] Environment variables accessible in Exec
- [ ] ExecWithTimeout respects timeout
- [ ] All unit tests pass
- [ ] No resource leaks (run tests multiple times)

## Success Criteria

- [ ] All advanced operations implemented
- [ ] Environment variable propagation working
- [ ] Tests pass with Docker running
- [ ] Resource cleanup verified (no leaked containers)
- [ ] Thread-safe with proper locking
- [ ] Error handling with proper context

## Integration Notes

These operations will be used by:
- **IO Provider**: RefreshFromHost for loading scaffolded projects
- **Validation**: WithEnvs for Databricks credentials
- **Artifact Collection**: ExportDirectory for logs/reports
- **Parallel Validation**: Fork for concurrent checks

## Next Steps

After completing this phase, proceed to:
- **parity-46**: Integration, Testing & Configuration
