package session

import (
	"context"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	s := NewSession()

	if s.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if s.workDir != "" {
		t.Error("workDir should be empty initially")
	}

	if !s.firstTool {
		t.Error("firstTool should be true initially")
	}

	if s.toolCalls != 0 {
		t.Error("toolCalls should be 0 initially")
	}
}

func TestSession_SetWorkDir(t *testing.T) {
	s := NewSession()
	ctx := WithSession(context.Background(), s)

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
	ctx := WithSession(context.Background(), s)

	_, err := GetWorkDir(ctx)
	if err == nil {
		t.Error("GetWorkDir should fail when work dir is not set")
	}
}

func TestSession_IsFirstTool(t *testing.T) {
	s := NewSession()

	// First call should return true
	if !s.IsFirstTool() {
		t.Error("First IsFirstTool call should return true")
	}

	// Subsequent calls should return false
	if s.IsFirstTool() {
		t.Error("Second IsFirstTool call should return false")
	}

	if s.IsFirstTool() {
		t.Error("Third IsFirstTool call should return false")
	}
}

func TestSession_ToolCalls(t *testing.T) {
	s := NewSession()

	if s.GetToolCalls() != 0 {
		t.Error("Initial tool calls should be 0")
	}

	count := s.IncrementToolCalls()
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	if s.GetToolCalls() != 1 {
		t.Errorf("Expected tool calls 1, got %d", s.GetToolCalls())
	}

	count = s.IncrementToolCalls()
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}

	if s.GetToolCalls() != 2 {
		t.Errorf("Expected tool calls 2, got %d", s.GetToolCalls())
	}
}

func TestSession_GetUptime(t *testing.T) {
	s := NewSession()

	// Sleep a bit to ensure uptime is measurable
	time.Sleep(10 * time.Millisecond)

	uptime := s.GetUptime()
	if uptime < 10*time.Millisecond {
		t.Errorf("Expected uptime >= 10ms, got %v", uptime)
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
