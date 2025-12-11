package fs

import (
	"context"
	"strings"
	"testing"
)

func TestCpConcurrencyValidation(t *testing.T) {
	ctx := context.Background()
	cmd := newCpCommand()
	cmd.SetContext(ctx)

	// Test concurrency = 0
	cmd.SetArgs([]string{"src", "dst", "--concurrency", "0"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--concurrency must be at least 1") {
		t.Errorf("expected error containing '--concurrency must be at least 1', got %v", err)
	}

	// Test concurrency = -1
	cmd.SetArgs([]string{"src", "dst", "--concurrency", "-1"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--concurrency must be at least 1") {
		t.Errorf("expected error containing '--concurrency must be at least 1', got %v", err)
	}
}
