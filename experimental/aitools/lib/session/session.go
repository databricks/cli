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
	WorkDirDataKey = "workDir"
)

// Session represents a CLI session with state tracking
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
