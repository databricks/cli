// Package dagger provides a stub implementation for Dagger-based sandbox.
// This is a placeholder for future containerized execution support.
package dagger

import (
	"context"
	"errors"

	"github.com/databricks/cli/experimental/apps-mcp/lib/sandbox"
)

func init() {
	sandbox.Register(sandbox.TypeDagger, func(cfg *sandbox.Config) (sandbox.Sandbox, error) {
		return nil, errors.New("dagger sandbox is not implemented")
	})
}

// DaggerSandbox is a stub implementation that always returns errors.
type DaggerSandbox struct{}

// Exec is not implemented.
func (d *DaggerSandbox) Exec(ctx context.Context, command string) (*sandbox.ExecResult, error) {
	return nil, errors.New("dagger sandbox is not implemented")
}

// WriteFile is not implemented.
func (d *DaggerSandbox) WriteFile(ctx context.Context, path, content string) error {
	return errors.New("dagger sandbox is not implemented")
}

// WriteFiles is not implemented.
func (d *DaggerSandbox) WriteFiles(ctx context.Context, files map[string]string) error {
	return errors.New("dagger sandbox is not implemented")
}

// ReadFile is not implemented.
func (d *DaggerSandbox) ReadFile(ctx context.Context, path string) (string, error) {
	return "", errors.New("dagger sandbox is not implemented")
}

// DeleteFile is not implemented.
func (d *DaggerSandbox) DeleteFile(ctx context.Context, path string) error {
	return errors.New("dagger sandbox is not implemented")
}

// ListDirectory is not implemented.
func (d *DaggerSandbox) ListDirectory(ctx context.Context, path string) ([]string, error) {
	return nil, errors.New("dagger sandbox is not implemented")
}

// SetWorkdir is not implemented.
func (d *DaggerSandbox) SetWorkdir(ctx context.Context, path string) error {
	return errors.New("dagger sandbox is not implemented")
}

// ExportDirectory is not implemented.
func (d *DaggerSandbox) ExportDirectory(ctx context.Context, containerPath, hostPath string) (string, error) {
	return "", errors.New("dagger sandbox is not implemented")
}

// RefreshFromHost is not implemented.
func (d *DaggerSandbox) RefreshFromHost(ctx context.Context, hostPath, containerPath string) error {
	return errors.New("dagger sandbox is not implemented")
}

// Close is not implemented.
func (d *DaggerSandbox) Close() error {
	return nil
}
