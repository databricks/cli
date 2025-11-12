package session

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// Session represents an MCP session with state tracking
type Session struct {
	ID        string
	WorkDir   string
	mu        sync.RWMutex
	startTime time.Time
	firstTool bool
	toolCalls int
	Tracker   interface{} // trajectory tracker (to avoid import cycle)
}

// NewSession creates a new session
func NewSession() *Session {
	return &Session{
		ID:        generateID(),
		startTime: time.Now(),
		firstTool: true,
	}
}

// SetWorkDir sets the working directory for this session
func (s *Session) SetWorkDir(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.WorkDir != "" {
		return errors.New("work directory already set")
	}

	s.WorkDir = dir
	return nil
}

// GetWorkDir returns the working directory for this session
func (s *Session) GetWorkDir() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.WorkDir == "" {
		return "", errors.New("work directory not set")
	}

	return s.WorkDir, nil
}

// IsFirstTool returns true if this is the first tool call in the session
// and sets the flag to false for subsequent calls
func (s *Session) IsFirstTool() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.firstTool {
		s.firstTool = false
		return true
	}
	return false
}

// IncrementToolCalls increments the tool call counter
func (s *Session) IncrementToolCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.toolCalls++
	return s.toolCalls
}

// GetToolCalls returns the number of tool calls made in this session
func (s *Session) GetToolCalls() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.toolCalls
}

// GetUptime returns the duration since the session started
func (s *Session) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// generateID generates a unique session ID
func generateID() string {
	return fmt.Sprintf("%d-%s", time.Now().Unix(), randomString(8))
}

// randomString generates a random string of the given length using crypto/rand
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			panic(fmt.Sprintf("crypto/rand failed: %v", err))
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
