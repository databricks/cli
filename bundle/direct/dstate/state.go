package dstate

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/internal/build"
	"github.com/google/uuid"
)

const (
	currentStateVersion = 2
	initialBufferSize   = 64 * 1024
	maxWalEntrySize     = 1024 * 1024
	walSuffix           = ".WAL"
)

type DeploymentState struct {
	Path     string
	Data     Database
	mu       sync.Mutex
	walFile  *os.File
	stateIDs map[string]string
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

type WALHeader struct {
	Lineage      string `json:"lineage"`
	Serial       int    `json:"serial"`
	StateVersion int    `json:"state_version"`
	CLIVersion   string `json:"cli_version"`
}

type WALEntry struct {
	Key   string         `json:"k"`
	Value *ResourceEntry `json:"v,omitempty"` // nil means delete
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
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		db.Data.State = make(map[string]ResourceEntry)
	}

	// don't indent so that every WAL entry remains on a single line
	jsonMessage, err := json.Marshal(state)
	if err != nil {
		return err
	}

	entry := ResourceEntry{
		ID:        newID,
		State:     json.RawMessage(jsonMessage),
		DependsOn: dependsOn,
	}

	err = appendJSONLine(db.walFile, WALEntry{Key: key, Value: &entry})
	if err == nil {
		db.stateIDs[key] = newID
	}
	return err
}

func (db *DeploymentState) DeleteState(key string) error {
	db.AssertOpenedForWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return nil
	}

	err := appendJSONLine(db.walFile, WALEntry{Key: key})
	if err == nil {
		delete(db.stateIDs, key)
	}
	return err
}

func (db *DeploymentState) GetResourceEntry(key string) (ResourceEntry, bool) {
	db.AssertOpenedForRead()
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Data.State == nil {
		return ResourceEntry{}, false
	}

	result, ok := db.Data.State[key]
	return result, ok
}

// GetResourceID returns the ID of the resource for the given key, or an empty string if not found.
func (db *DeploymentState) GetResourceID(key string) string {
	db.AssertOpenedForReadOrWrite()
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.stateIDs[key]
}

type (
	WithRecovery bool
	WithWrite    bool
)

func (db *DeploymentState) Open(ctx context.Context, path string, withRecovery WithRecovery, withWrite WithWrite) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	db.Path = path
	if err := db.Reload(ctx); err != nil {
		return err
	}

	walPath := db.Path + walSuffix
	_, walError := os.Stat(walPath)
	if walError == nil {
		if withRecovery {
			err := db.replayWAL(ctx)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unexpected WAL file found at %s", walPath)
		}
	}

	if err := migrateState(&db.Data); err != nil {
		return fmt.Errorf("migrating state %s: %w", path, err)
	}

	if withWrite {
		walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
		}
		db.walFile = walFile
		lineage := db.Data.Lineage
		if lineage == "" {
			// state file is new, does not have lineage yet; store lineage in the WAL only
			lineage = uuid.New().String()
		}
		walHead := WALHeader{
			Lineage:      lineage,
			Serial:       db.Data.Serial + 1,
			StateVersion: currentStateVersion,
			CLIVersion:   build.GetInfo().Version,
		}
		return appendJSONLine(db.walFile, walHead)
	}

	return nil
}

func (db *DeploymentState) Reload(ctx context.Context) error {
	db.stateIDs = make(map[string]string)
	data, err := os.ReadFile(db.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Not initializing lineage yet, we might have that saved in WAL
			db.Data = NewDatabase("", 0)
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &db.Data); err != nil {
			return err
		}
	}
	for key, entry := range db.Data.State {
		db.stateIDs[key] = entry.ID
	}
	return nil
}

func (db *DeploymentState) replayWAL(ctx context.Context) error {
	walPath := db.Path + walSuffix
	hasEntries, err := db.mergeWalIntoState(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply WAL file %s: %w", walPath, err)
	}
	if hasEntries {
		if err := db.unlockedSave(); err != nil {
			return err
		}
	}
	err = os.Remove(walPath)
	if err != nil {
		return fmt.Errorf("failed to remove WAL file %s: %w", walPath, err)
	}
	return nil
}

func (db *DeploymentState) validateWALHeader(ctx context.Context, header *WALHeader) error {
	if header.CLIVersion != db.Data.CLIVersion {
		return fmt.Errorf("cli_version in the header (%q) does not match the one in the state (%q)", header.CLIVersion, db.Data.CLIVersion)
	}

	if header.StateVersion != db.Data.StateVersion {
		return fmt.Errorf("state_version in the header (%d) does not match the one in the state (%d)", header.StateVersion, db.Data.StateVersion)
	}

	if header.Lineage != db.Data.Lineage && db.Data.Lineage != "" {
		return fmt.Errorf("lineage in the header (%q) does not match the one in the state (%q)", header.Lineage, db.Data.Lineage)
	}

	if header.Serial != db.Data.Serial+1 {
		return fmt.Errorf("serial in the header (%d) is not one higher than the one in the state (%d)", header.Serial, db.Data.Serial)
	}

	return nil
}

func (db *DeploymentState) mergeWalIntoState(ctx context.Context) (bool, error) {
	if db.walFile != nil {
		panic("internal error: walFile must be closed")
	}

	walPath := db.Path + walSuffix
	walFile, err := os.Open(walPath)
	if err != nil {
		return false, fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
	}
	defer walFile.Close()

	scanner := bufio.NewScanner(walFile)
	scanner.Buffer(make([]byte, 0, initialBufferSize), maxWalEntrySize)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if lineNumber == 1 {
			var header WALHeader
			if err := json.Unmarshal(line, &header); err != nil {
				return false, fmt.Errorf("failed to parse WAL header: %w", err)
			}
			if err := db.validateWALHeader(ctx, &header); err != nil {
				return false, err
			}
			// Apply header metadata to state (lineage may be new for first deploy)
			db.Data.Lineage = header.Lineage
			db.Data.Serial = header.Serial
		} else {
			var entry WALEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				return false, fmt.Errorf("failed to parse WAL entry %s:%d: %q: %w", walPath, lineNumber, line, err)
			}
			if entry.Value == nil {
				delete(db.Data.State, entry.Key)
			} else {
				db.Data.State[entry.Key] = *entry.Value
			}
		}
	}

	return lineNumber > 1, scanner.Err()
}

// Close replays the WAL (if open for write) and resets the state.
// Safe to call multiple times or on an already-closed state.
func (db *DeploymentState) Close(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path == "" {
		return nil
	}

	var err error

	if db.walFile != nil {
		db.walFile.Close()
		db.walFile = nil
		err = db.replayWAL(ctx)
	}

	db.Path = ""
	db.Data = Database{}
	db.stateIDs = make(map[string]string)

	return err
}

// UpgradeToWrite transitions from read mode to write mode without re-reading state.
// State must already be open for read. This initializes the WAL for writing.
func (db *DeploymentState) UpgradeToWrite() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path == "" {
		return errors.New("internal error: DeploymentState must be opened first")
	}
	if db.walFile != nil {
		return errors.New("internal error: DeploymentState is already open for write")
	}

	walPath := db.Path + walSuffix
	walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
	}
	db.walFile = walFile

	lineage := db.Data.Lineage
	if lineage == "" {
		lineage = uuid.New().String()
	}
	walHead := WALHeader{
		Lineage:      lineage,
		Serial:       db.Data.Serial + 1,
		StateVersion: currentStateVersion,
		CLIVersion:   build.GetInfo().Version,
	}
	return appendJSONLine(db.walFile, walHead)
}

func (db *DeploymentState) AssertOpenedForReadOrWrite() {
	if db.Path == "" {
		panic("internal error: DeploymentState must be opened first")
	}
}

func (db *DeploymentState) AssertOpenedForRead() {
	db.AssertOpenedForReadOrWrite()
	if db.walFile != nil {
		panic("internal error: DeploymentState must be opened in read mode")
	}
}

func (db *DeploymentState) AssertOpenedForWrite() {
	db.AssertOpenedForReadOrWrite()
	if db.walFile == nil {
		panic("internal error: DeploymentState must be opened in write mode")
	}
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
	data, err := json.MarshalIndent(db.Data, "", " ")
	if err != nil {
		return err
	}

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

func appendJSONLine(file *os.File, obj any) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = file.Write(data)
	// no fsync here, not needed
	return err
}
