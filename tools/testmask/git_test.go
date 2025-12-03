package main

import (
	"testing"
)

func TestGetChangedFiles(t *testing.T) {
	// Test with HEAD to HEAD - should return empty list
	result, err := GetChangedFiles("HEAD", "HEAD")
	if err != nil {
		t.Skipf("unable to run git: %q", err)
		return
	}
	if len(result) > 0 {
		t.Errorf("expected empty list, got %q", result)
	}

	// Test with HEAD to HEAD~2 - should produce non-empty result if there are commits
	result, err = GetChangedFiles("HEAD", "HEAD~2")
	if err != nil {
		t.Errorf("unable to run git: %q", err)
		return
	}
	if len(result) == 0 {
		t.Errorf("expected non-empty list, got %q", result)
	}

	// Test with invalid refs - should error
	_, err = GetChangedFiles("invalid-ref-12345", "invalid-ref-67890")
	if err == nil {
		t.Error("expected error for invalid refs")
	}
}
