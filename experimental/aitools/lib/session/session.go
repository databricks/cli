package session

import (
	"context"
	"errors"
	"sync"
)

type sessionKey struct{}

// Session represents a CLI session with state tracking
type Session struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewSession creates a new session
func NewSession() *Session {
	return &Session{
		data: make(map[string]any),
	}
}

// WithSession adds session to context
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// GetSession retrieves session from context
func GetSession(ctx context.Context) (*Session, error) {
	if v := ctx.Value(sessionKey{}); v != nil {
		return v.(*Session), nil
	}
	return nil, errors.New("session not found in context")
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

// Set stores a value in session data.
func (s *Session) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}
