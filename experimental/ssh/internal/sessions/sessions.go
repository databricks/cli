package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

const (
	stateFileName = "ssh-tunnel-sessions.json"
	lockFileName  = "ssh-tunnel-sessions.lock"

	// Sessions older than this are considered expired and cleaned up automatically.
	sessionMaxAge = 24 * time.Hour

	// Lock acquisition parameters.
	lockRetryInterval = 100 * time.Millisecond
	lockTimeout       = 5 * time.Second
	// Locks older than this are considered stale and can be broken.
	lockMaxAge = 30 * time.Second
)

// Session represents a tracked SSH tunnel session.
type Session struct {
	Name          string    `json:"name"`
	Accelerator   string    `json:"accelerator"`
	WorkspaceHost string    `json:"workspace_host"`
	UserName      string    `json:"user_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	ClusterID     string    `json:"cluster_id,omitempty"`
}

// SessionStore holds all tracked sessions.
type SessionStore struct {
	Sessions []Session `json:"sessions"`
}

func getStateDir(ctx context.Context) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".databricks"), nil
}

func getStateFilePath(ctx context.Context) (string, error) {
	dir, err := getStateDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

// acquireLock acquires an exclusive file lock for the session store.
// Returns an unlock function that must be called when done.
func acquireLock(ctx context.Context) (unlock func(), err error) {
	dir, err := getStateDir(ctx)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	lockPath := filepath.Join(dir, lockFileName)
	deadline := time.Now().Add(lockTimeout)

	for {
		f, createErr := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr == nil {
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}

		// Break stale locks from crashed processes.
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > lockMaxAge {
			log.Debugf(ctx, "Breaking stale session lock (age: %v)", time.Since(info.ModTime()))
			os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for session store lock")
		}
		time.Sleep(lockRetryInterval)
	}
}

// Load reads the session store from disk. Returns an empty store if the file does not exist.
func Load(ctx context.Context) (*SessionStore, error) {
	path, err := getStateFilePath(ctx)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &SessionStore{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read session state file: %w", err)
	}

	var store SessionStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse session state file: %w", err)
	}
	return &store, nil
}

// save writes the session store to disk atomically. Caller must hold the lock.
func save(ctx context.Context, store *SessionStore) error {
	path, err := getStateFilePath(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}

	// Atomic write: write to unique temp file, then rename.
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".sessions-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write session state file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename session state file: %w", err)
	}
	return nil
}

// Add persists a new session to the store, replacing any existing session with the same name.
func Add(ctx context.Context, s Session) error {
	unlock, err := acquireLock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire session store lock: %w", err)
	}
	defer unlock()

	store, err := Load(ctx)
	if err != nil {
		return err
	}

	// Replace existing session with the same name.
	found := false
	for i, existing := range store.Sessions {
		if existing.Name == s.Name {
			store.Sessions[i] = s
			found = true
			break
		}
	}
	if !found {
		store.Sessions = append(store.Sessions, s)
	}

	return save(ctx, store)
}

// Remove deletes a session by name.
func Remove(ctx context.Context, name string) error {
	unlock, err := acquireLock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire session store lock: %w", err)
	}
	defer unlock()

	store, err := Load(ctx)
	if err != nil {
		return err
	}

	filtered := store.Sessions[:0]
	for _, s := range store.Sessions {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	store.Sessions = filtered
	return save(ctx, store)
}

// FindMatching returns non-expired sessions that match the given workspace host, accelerator,
// and user name. Expired sessions are pruned from the store on disk.
func FindMatching(ctx context.Context, workspaceHost, accelerator, userName string) ([]Session, error) {
	unlock, err := acquireLock(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire session store lock: %w", err)
	}
	defer unlock()

	store, err := Load(ctx)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-sessionMaxAge)

	// Prune expired sessions from the store.
	active := store.Sessions[:0]
	pruned := false
	for _, s := range store.Sessions {
		if s.CreatedAt.After(cutoff) {
			active = append(active, s)
		} else {
			pruned = true
		}
	}
	if pruned {
		store.Sessions = active
		_ = save(ctx, store)
	}

	var result []Session
	for _, s := range active {
		if s.WorkspaceHost == workspaceHost && s.Accelerator == accelerator && s.UserName == userName {
			result = append(result, s)
		}
	}
	return result, nil
}
