package main

import (
	"testing"
)

// gitEmptyTreeSHA is the well-known SHA of the empty tree object. Git resolves
// it without requiring a commit to reference it, so diffing against it works
// even on shallow clones where HEAD~N is unavailable (as in CI).
const gitEmptyTreeSHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

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

	// Test against the empty tree - should produce non-empty result (all files in HEAD)
	result, err = GetChangedFiles("HEAD", gitEmptyTreeSHA)
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
