package main

import (
	"testing"
)

func TestParseLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "multiple lines",
			input:    "file1.go\nfile2.go\nfile3.go\n",
			expected: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:     "whitespace trimmed and empty lines ignored",
			input:    "  file1.go  \n\nfile2.go\n\t\n",
			expected: []string{"file1.go", "file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLines([]byte(tt.input))
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d lines, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("line[%d]: expected %q, got %q", i, tt.expected[i], line)
				}
			}
		})
	}
}

func TestGetChangedFiles(t *testing.T) {
	// Test with HEAD to HEAD - should return empty list
	result, err := GetChangedFiles("HEAD", "HEAD")
	if err != nil {
		t.Skipf("unable to run git: %v", err)
		return
	}
	if len(result) > 0 {
		t.Errorf("expected empty list, got %v", result)
	}

	// Test with HEAD to HEAD~2 - should produce non-empty result if there are commits
	result, err = GetChangedFiles("HEAD", "HEAD~2")
	if err != nil {
		t.Errorf("unable to run git: %v", err)
		return
	}
	if len(result) == 0 {
		t.Errorf("expected non-empty list, got %v", result)
	}

	// Test with invalid refs - should error
	_, err = GetChangedFiles("invalid-ref-12345", "invalid-ref-67890")
	if err == nil {
		t.Error("expected error for invalid refs")
	}
}
