package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/google/uuid"
)

const currentStateVersion = 1

type DeploymentState struct {
	Path             string
	Data             Database
	mu               sync.Mutex
	wal              *WAL
	recoveredFromWAL bool
}

type Database struct {
	StateVersion int                      `json:"state_version"`
	CLIVersion   string                   `json:"cli_version"`
	Lineage      string                   `json:"lineage"`
	Serial       int                      `json:"serial"`
	State        map[string]ResourceEntry `json:"state"`
}

type ResourceEntry struct {
	ID        string                      `json:"__id__"`
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

func NewDatabase(lineage string, serial int) Database {
	return Database{
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
		Lineage:      lineage,
		Serial:       serial,
		State:        make(map[string]ResourceEntry),
	}
}

func (db *DeploymentState) SaveState(key, newID string, state any, dependsOn []deployplan.DependsOnEntry) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	jsonMessage, err := json.MarshalIndent(state, "", " ")
	if err != nil {
		return err
	}

	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	if err := db.ensureWALOpen(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	if err := db.wal.writeEntry(key, &entry); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	db.Data.State[key] = entry

	return nil
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}

	if err := db.ensureWALOpen(); err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	if err := db.wal.writeEntry(key, nil); err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	delete(db.Data.State, key)

	return nil
}

// ensureWALOpen opens the WAL file and writes the header if not already done.
// Must be called while holding db.mu.
func (db *DeploymentState) ensureWALOpen() error {
	if db.wal != nil {
		return nil
	}

	wal, err := openWAL(db.Path)
	if err != nil {
		return err
	}

	lineage := db.Data.Lineage
	if lineage == "" {
		lineage = uuid.New().String()
		db.Data.Lineage = lineage
	}

	// WAL serial is the NEXT serial (current + 1)
	walSerial := db.Data.Serial + 1

	if err := wal.writeHeader(lineage, walSerial); err != nil {
		wal.close()
		return err
	}

	db.wal = wal
	return nil
}

func (db *DeploymentState) GetResourceEntry(key string) (ResourceEntry, bool) {
	db.AssertOpened()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return ResourceEntry{}, false
	}

	result, ok := db.Data.State[key]
	return result, ok
}

func (db *DeploymentState) Open(ctx context.Context, path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		if db.Path == path {
			return nil
		}
		return fmt.Errorf("already read state %v, cannot open %v", db.Path, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create new database with serial=0, will be incremented to 1 in Finalize()
			db.Data = NewDatabase("", 0)
			db.Path = path

			// Write state file immediately to ensure it exists before any WAL operations.
			// This guarantees we have a base state file for recovery validation.
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("failed to create state directory: %w", err)
			}
			if err := db.unlockedSave(); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		err = json.Unmarshal(data, &db.Data)
		if err != nil {
			return err
		}
		db.Path = path
	}

	recovered, err := recoverFromWAL(ctx, path, &db.Data)
	if err != nil {
		return fmt.Errorf("WAL recovery failed: %w", err)
	}
	if recovered {
		log.Infof(ctx, "Recovered deployment state from WAL")
		db.recoveredFromWAL = true
	}

	return nil
}

func (db *DeploymentState) Finalize() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Generate lineage on first save (if WAL wasn't opened)
	if db.Data.Lineage == "" {
		db.Data.Lineage = uuid.New().String()
	}

	db.Data.Serial++

	err := db.unlockedSave()
	if err != nil {
		return err
	}

	if db.wal != nil {
		if err := db.wal.truncate(); err != nil {
			return fmt.Errorf("failed to truncate WAL: %w", err)
		}
		db.wal = nil
	} else {
		// No WAL was opened, but we should still clean up any stale WAL file
		wp := walPath(db.Path)
		if err := os.Remove(wp); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove stale WAL file: %w", err)
		}
	}

	return nil
}

// Close closes the WAL file handle without finalizing or truncating.
// Use this in tests or when you need to abort without saving state.
func (db *DeploymentState) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.wal != nil {
		if err := db.wal.close(); err != nil {
			return err
		}
		db.wal = nil
	}
	return nil
}

func (db *DeploymentState) AssertOpened() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

// RecoveredFromWAL returns true if state was recovered from WAL during Open().
// This is used to determine if Finalize() should be called even with an empty plan.
func (db *DeploymentState) RecoveredFromWAL() bool {
	return db.recoveredFromWAL
}

func (db *DeploymentState) ExportState(ctx context.Context) resourcestate.ExportedResourcesMap {
	result := make(resourcestate.ExportedResourcesMap)
	for key, entry := range db.Data.State {
		var etag string
		// Extract etag for dashboards.
		// covered by test case: bundle/deploy/dashboard/detect-change
		if strings.Contains(key, ".dashboards.") && len(entry.State) > 0 {
			var holder struct {
				Etag string `json:"etag"`
			}
			if err := json.Unmarshal(entry.State, &holder); err == nil {
				etag = holder.Etag
			}
		}

		result[key] = resourcestate.ResourceState{
			ID:   entry.ID,
			ETag: etag,
		}
	}
	return result
}

func (db *DeploymentState) unlockedSave() error {
	db.AssertOpened()
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(db.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %#v: %w", dir, err)
	}

	err = os.WriteFile(db.Path, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to save resources state to %#v: %w", db.Path, err)
	}

	return nil
}
