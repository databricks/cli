package session

import (
	"testing"
)

func TestNewSession(t *testing.T) {
	s := NewSession()

	if s.ID == "" {
		t.Error("Session ID should not be empty")
	}

	// Check data fields using Get
	if workDir, ok := s.Get(WorkDirDataKey); ok && workDir != nil && workDir != "" {
		t.Error("workDir should be empty initially")
	}
}

func TestSession_SetWorkDir(t *testing.T) {
	s := NewSession()
	ctx := WithSession(t.Context(), s)

	// First set should succeed
	err := SetWorkDir(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("First SetWorkDir failed: %v", err)
	}

	// Second set should fail
	err = SetWorkDir(ctx, "/tmp/test2")
	if err == nil {
		t.Error("Second SetWorkDir should fail")
	}

	// Verify the work dir is set correctly
	workDir, err := GetWorkDir(ctx)
	if err != nil {
		t.Fatalf("GetWorkDir failed: %v", err)
	}

	if workDir != "/tmp/test" {
		t.Errorf("Expected work dir '/tmp/test', got '%s'", workDir)
	}
}

func TestSession_GetWorkDir_NotSet(t *testing.T) {
	s := NewSession()
	ctx := WithSession(t.Context(), s)

	_, err := GetWorkDir(ctx)
	if err == nil {
		t.Error("GetWorkDir should fail when work dir is not set")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}
}

func TestRandomString(t *testing.T) {
	s1 := randomString(8)
	s2 := randomString(8)

	if len(s1) != 8 {
		t.Errorf("Expected length 8, got %d", len(s1))
	}

	if len(s2) != 8 {
		t.Errorf("Expected length 8, got %d", len(s2))
	}

	if s1 == s2 {
		t.Error("Random strings should be different")
	}
}
