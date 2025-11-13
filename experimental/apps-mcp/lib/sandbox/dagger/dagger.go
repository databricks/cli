// Package dagger provides a Dagger-based sandbox implementation for
// containerized execution and file operations.
package dagger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dagger.io/dagger"
	"github.com/databricks/cli/experimental/apps-mcp/lib/sandbox"
)

var (
	globalClient *dagger.Client
	clientMu     sync.Mutex
)

func init() {
	sandbox.Register(sandbox.TypeDagger, func(cfg *sandbox.Config) (sandbox.Sandbox, error) {
		return NewDaggerSandbox(context.Background(), Config{
			Image:          "node:20-alpine",
			ExecuteTimeout: int(cfg.Timeout.Seconds()),
			BaseDir:        cfg.BaseDir,
		})
	})
}

// GetGlobalClient returns a singleton Dagger client, creating it if necessary.
// This enables connection pooling across multiple sandbox instances.
func GetGlobalClient(ctx context.Context) (*dagger.Client, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if globalClient == nil {
		var err error
		globalClient, err = dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Dagger: %w", err)
		}
	}

	return globalClient, nil
}

// CloseGlobalClient closes the global Dagger client if it exists.
// Should be called during application shutdown.
func CloseGlobalClient() error {
	clientMu.Lock()
	defer clientMu.Unlock()

	if globalClient != nil {
		err := globalClient.Close()
		globalClient = nil
		return err
	}

	return nil
}

// DaggerSandbox implements the Sandbox interface using Dagger containers.
// It provides isolated execution environments with container-based operations.
type DaggerSandbox struct {
	client    *dagger.Client
	container *dagger.Container
	workdir   string
	baseDir   string
	mu        sync.RWMutex

	image          string
	executeTimeout int
}

// Config holds configuration options for creating a DaggerSandbox.
type Config struct {
	// Image is the Docker image to use (default: node:20-alpine)
	Image string
	// ExecuteTimeout is the execution timeout in seconds (default: 600)
	ExecuteTimeout int
	// BaseDir is the base directory for operations
	BaseDir string
}

// NewDaggerSandbox creates a new DaggerSandbox with the specified configuration.
// It uses the global Dagger client for connection pooling and initializes a container
// with the specified image.
func NewDaggerSandbox(ctx context.Context, cfg Config) (*DaggerSandbox, error) {
	if cfg.Image == "" {
		cfg.Image = "node:20-alpine"
	}
	if cfg.ExecuteTimeout == 0 {
		cfg.ExecuteTimeout = 600
	}

	client, err := GetGlobalClient(ctx)
	if err != nil {
		return nil, err
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

// resolvePath resolves a path relative to the working directory.
// If the path is absolute, it returns it as-is.
// If the path is relative, it joins it with the working directory.
func (d *DaggerSandbox) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(d.workdir, path)
}

// Exec executes a command in the Dagger container.
// The command is executed using "sh -c" to support shell features.
func (d *DaggerSandbox) Exec(ctx context.Context, command string) (*sandbox.ExecResult, error) {
	d.mu.RLock()
	container := d.container.WithExec([]string{"sh", "-c", command}, dagger.ContainerWithExecOpts{
		Expect: dagger.ReturnTypeAny,
	})
	d.mu.RUnlock()

	exitCode, exitErr := container.ExitCode(ctx)
	if exitErr != nil {
		return nil, fmt.Errorf("failed to get exit code: %w", exitErr)
	}

	stdout, stdoutErr := container.Stdout(ctx)
	if stdoutErr != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", stdoutErr)
	}

	stderr, stderrErr := container.Stderr(ctx)
	if stderrErr != nil {
		return nil, fmt.Errorf("failed to get stderr: %w", stderrErr)
	}

	result := &sandbox.ExecResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}

	d.mu.Lock()
	d.container = container
	d.mu.Unlock()

	if exitCode != 0 {
		return result, fmt.Errorf("command exited with code %d", exitCode)
	}

	return result, nil
}

// WriteFile writes content to a file in the container.
// Parent directories are created automatically if they don't exist.
func (d *DaggerSandbox) WriteFile(ctx context.Context, path, content string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	fullPath := d.resolvePath(path)
	dir := filepath.Dir(fullPath)

	d.container = d.container.WithExec([]string{"mkdir", "-p", dir})
	d.container = d.container.WithNewFile(fullPath, content, dagger.ContainerWithNewFileOpts{
		Permissions: 0o644,
	})

	_, err := d.container.Sync(ctx)
	return err
}

// WriteFiles writes multiple files to the container in a single operation.
// This is much more efficient than individual WriteFile calls as it prevents
// deep query chains in Dagger.
func (d *DaggerSandbox) WriteFiles(ctx context.Context, files map[string]string) error {
	if len(files) == 0 {
		return nil
	}

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

// ReadFile reads the content of a file from the container.
func (d *DaggerSandbox) ReadFile(ctx context.Context, path string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	fullPath := d.resolvePath(path)

	file := d.container.File(fullPath)
	contents, err := file.Contents(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return contents, nil
}

// DeleteFile deletes a file from the container.
func (d *DaggerSandbox) DeleteFile(ctx context.Context, path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	fullPath := d.resolvePath(path)
	d.container = d.container.WithExec([]string{"rm", "-f", fullPath})

	_, err := d.container.Sync(ctx)
	return err
}

// ListDirectory lists all files and directories in the specified path.
// Returns a list of entry names (not full paths).
func (d *DaggerSandbox) ListDirectory(ctx context.Context, path string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	fullPath := d.resolvePath(path)

	dir := d.container.Directory(fullPath)
	entries, err := dir.Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory %s: %w", path, err)
	}

	return entries, nil
}

// SetWorkdir changes the working directory for future operations.
func (d *DaggerSandbox) SetWorkdir(ctx context.Context, path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.workdir = path
	d.container = d.container.WithWorkdir(path)

	return nil
}

// ExportDirectory exports a directory from the container to the host filesystem.
// Returns the absolute path to the exported directory on the host.
func (d *DaggerSandbox) ExportDirectory(ctx context.Context, containerPath, hostPath string) (string, error) {
	d.mu.RLock()
	dir := d.container.Directory(containerPath)
	d.mu.RUnlock()

	exportedPath, err := dir.Export(ctx, hostPath)
	if err != nil {
		return "", fmt.Errorf("failed to export directory: %w", err)
	}

	return exportedPath, nil
}

// RefreshFromHost imports files from the host filesystem into the container.
// This is useful for incremental updates without recreating the sandbox.
func (d *DaggerSandbox) RefreshFromHost(ctx context.Context, hostPath, containerPath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	hostDir := d.client.Host().Directory(hostPath)
	d.container = d.container.WithDirectory(containerPath, hostDir)

	_, err := d.container.Sync(ctx)
	return err
}

// Close releases the Dagger client resources.
// After calling Close, the sandbox should not be used.
func (d *DaggerSandbox) Close() error {
	// With connection pooling, individual sandbox instances do not close
	// the global client. Use CloseGlobalClient() during application shutdown.
	// We only clear the container reference, not the client.
	d.mu.Lock()
	d.container = nil
	d.mu.Unlock()
	return nil
}

// Fork creates a copy of the sandbox with the same state.
// This is lightweight due to Dagger's immutability model - containers
// are just references that can be safely shared.
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

// WithEnv sets a single environment variable in the container.
// This modifies the container state, so subsequent operations will have access to it.
func (d *DaggerSandbox) WithEnv(key, value string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.container = d.container.WithEnvVariable(key, value)
}

// WithEnvs sets multiple environment variables in the container.
// This is more efficient than calling WithEnv multiple times separately.
func (d *DaggerSandbox) WithEnvs(envs map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for key, value := range envs {
		d.container = d.container.WithEnvVariable(key, value)
	}
}

// ExecWithTimeout executes a command with a timeout.
// If the timeout is exceeded, it returns context.DeadlineExceeded.
func (d *DaggerSandbox) ExecWithTimeout(ctx context.Context, command string) (*sandbox.ExecResult, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(d.executeTimeout)*time.Second)
	defer cancel()

	return d.Exec(timeoutCtx, command)
}
