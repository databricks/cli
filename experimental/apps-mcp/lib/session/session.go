package session

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// contextKey is the type for context keys
type contextKey int

const (
	workDirKey contextKey = iota
	sessionKey
)

// Data keys for session storage
const (
	WorkDirDataKey   = "workDir"
	StartTimeDataKey = "startTime"
	ToolCallsDataKey = "toolCalls"
	TrackerDataKey   = "tracker"
)

// Session represents an MCP session with state tracking
type Session struct {
	ID   string
	mu   sync.RWMutex
	data map[string]any
}

// NewSession creates a new session
func NewSession() *Session {
	sess := &Session{
		ID:   generateID(),
		data: make(map[string]any),
	}
	sess.data[StartTimeDataKey] = time.Now()
	sess.data[ToolCallsDataKey] = 0
	return sess
}

// WithSession adds session to context
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// GetSession retrieves session from context
func GetSession(ctx context.Context) (*Session, error) {
	if v := ctx.Value(sessionKey); v != nil {
		return v.(*Session), nil
	}
	return nil, errors.New("session not found in context")
}

// SetWorkDir sets the working directory via context
func SetWorkDir(ctx context.Context, dir string) error {
	sess, err := GetSession(ctx)
	if err != nil {
		return err
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if workDir, ok := sess.data[WorkDirDataKey]; ok && workDir != "" {
		return errors.New("work directory already set")
	}

	sess.data[WorkDirDataKey] = dir
	return nil
}

// GetWorkDir retrieves work directory via context
func GetWorkDir(ctx context.Context) (string, error) {
	sess, err := GetSession(ctx)
	if err != nil {
		return "", err
	}

	sess.mu.RLock()
	defer sess.mu.RUnlock()

	workDir, ok := sess.data[WorkDirDataKey]
	if !ok || workDir == "" {
		return "", errors.New("work directory not set")
	}

	return workDir.(string), nil
}

// IncrementToolCalls increments the tool call counter
func (s *Session) IncrementToolCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count, ok := s.data[ToolCallsDataKey]
	if !ok {
		count = 0
	}
	newCount := count.(int) + 1
	s.data[ToolCallsDataKey] = newCount
	return newCount
}

// GetToolCalls returns the number of tool calls made in this session
func (s *Session) GetToolCalls() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count, ok := s.data[ToolCallsDataKey]
	if !ok {
		return 0
	}
	return count.(int)
}

// GetUptime returns the duration since the session started
func (s *Session) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	startTime, ok := s.data[StartTimeDataKey]
	if !ok {
		return 0
	}
	return time.Since(startTime.(time.Time))
}

// SetTracker stores the trajectory tracker in the session
func (s *Session) SetTracker(tracker any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[TrackerDataKey] = tracker
}

// GetTracker retrieves the trajectory tracker from the session
func (s *Session) GetTracker() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[TrackerDataKey]
}

// Get retrieves a value from session data.
func (s *Session) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	valRaw, ok := s.data[key]
	if !ok {
		return nil, ok
	}
	return valRaw, true
}

// GetBool retrieves a value from session data and casts it to a boolean.
func (s *Session) GetBool(key string, defaultValue bool) bool {
	if val, ok := s.Get(key); ok {
		return val.(bool)
	}
	return defaultValue
}

// Set stores a value in session data.
func (s *Session) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Delete removes a value from session data.
func (s *Session) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
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
