package dstate

import (
	"bufio"
	"bytes"
	"compress/flate"
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
	"github.com/databricks/cli/libs/log"
	"github.com/google/uuid"
)

// compressedStateSize returns the DEFLATE-compressed size in bytes of a
// resource's serialized state. It is a rough proxy, used purely for deploy
// telemetry, for what the state sizes look like on the server side (which
// compresses with zstd): we deliberately use the standard library's
// compress/flate rather than pull in a dedicated zstd dependency, keeping the
// supply chain small while still getting useful signal on compressibility.
// Returns 0 for empty state.
//
// This is cheap: it runs at state-export time over individually small resource
// states (each well under the server's per-resource limit), not in a tight
// loop, so even large bundles compress in a few milliseconds — negligible next
// to the network I/O of a deploy. A background goroutine is not warranted.
func compressedStateSize(state []byte) int {
	if len(state) == 0 {
		return 0
	}
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return 0
	}
	if _, err := w.Write(state); err != nil {
		return 0
	}
	if err := w.Close(); err != nil {
		return 0
	}
	return buf.Len()
}

const (
	currentStateVersion = 2
	initialBufferSize   = 64 * 1024
	maxWalEntrySize     = 10 * 1024 * 1024
	walSuffix           = ".wal"
)

// errStaleWAL is returned when the WAL serial is behind the expected serial.
// The caller should delete the stale WAL and proceed normally.
var errStaleWAL = errors.New("stale WAL")

type DeploymentState struct {
	Path    string
	Data    Database
	mu      sync.Mutex
	walFile *os.File

	// Maps resource key to ID. Unlike Data.State, this is up to date during writes (deploys).
	stateIDs map[string]string
}

type Header struct {
	StateVersion int    `json:"state_version"`
	CLIVersion   string `json:"cli_version"`
	Lineage      string `json:"lineage"`
	Serial       int    `json:"serial"`
}

type Database struct {
	Header

	// Maps resource key to ResourceEntry which includes ID + full serialized state.
	// This is not updated during write/deploy, those writes go to WAL instead.
	// The State is then reconstructed from WAL.
	State map[string]ResourceEntry `json:"state"`
}

type ResourceEntry struct {
	ID        string                      `json:"__id__"`
	State     json.RawMessage             `json:"state"`
	DependsOn []deployplan.DependsOnEntry `json:"depends_on,omitempty"`
}

type WALEntry struct {
	Key   string         `json:"k"`
	Value *ResourceEntry `json:"v,omitempty"` // nil means delete
}

func NewDatabase(lineage string, serial int) Database {
	return Database{
		Header: Header{
			StateVersion: currentStateVersion,
			CLIVersion:   build.GetInfo().Version,
			Lineage:      lineage,
			Serial:       serial,
		},
		State: make(map[string]ResourceEntry),
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
	// Note, if opened for write, you get the state that you had at the beginning of deploy, not most recent one
	db.AssertOpenedForReadOrWrite()
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
	// If true, then Open reads the WAL and merges it in the state. If false, and WAL is present, Open returns an error.
	WithRecovery bool

	// If true, the state is opened in Write mode, which enables methods such as SaveState
	// but disables GetResourceEntry (since writes go strictly into WAL and not in memory).
	WithWrite bool
)

func (db *DeploymentState) Open(ctx context.Context, path string, withRecovery WithRecovery, withWrite WithWrite) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	db.Path = path
	data, err := os.ReadFile(db.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			db.Data = NewDatabase("", 0)
		} else {
			return err
		}
	} else {
		if err := json.Unmarshal(data, &db.Data); err != nil {
			return err
		}
	}

	db.stateIDs = make(map[string]string)
	for key, entry := range db.Data.State {
		db.stateIDs[key] = entry.ID
	}

	walPath := db.Path + walSuffix
	_, err = os.Stat(walPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// no WAL, nothing to do
	case err != nil:
		return fmt.Errorf("failed to stat WAL file %s: %w", walPath, err)
	default: // WAL exists
		if withRecovery {
			if err := db.replayWAL(ctx); err != nil {
				return fmt.Errorf("reading state from %s: %w", path, err)
			}
		} else {
			return fmt.Errorf("unexpected WAL file found at %s", walPath)
		}
	}

	if err := migrateState(&db.Data); err != nil {
		return fmt.Errorf("migrating state %s: %w", path, err)
	}

	if withWrite {
		if err := os.MkdirAll(filepath.Dir(walPath), 0o755); err != nil {
			return fmt.Errorf("failed to create state directory: %w", err)
		}
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
		walHead := Header{
			Lineage:      lineage,
			Serial:       db.Data.Serial + 1,
			StateVersion: currentStateVersion,
			CLIVersion:   build.GetInfo().Version,
		}
		return appendJSONLine(db.walFile, walHead)
	}

	return nil
}

// OpenWithData initializes the state from an in-memory database without reading from disk.
// The state is opened in read mode; call UpgradeToWrite to transition to write mode.
func (db *DeploymentState) OpenWithData(path string, data Database) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path != "" {
		panic(fmt.Sprintf("state already opened: %v, cannot open %v", db.Path, path))
	}

	db.Path = path
	db.Data = data
	db.stateIDs = make(map[string]string)
	for key, entry := range data.State {
		db.stateIDs[key] = entry.ID
	}
}

func (db *DeploymentState) replayWAL(ctx context.Context) error {
	walPath := db.Path + walSuffix
	hasEntries, err := db.mergeWalIntoState(ctx)
	if err != nil {
		if errors.Is(err, errStaleWAL) {
			log.Debugf(ctx, "Deleting stale WAL file %s", walPath)
			_ = os.Remove(walPath)
			return nil
		}
		return fmt.Errorf("WAL recovery failed: %w", err)
	}
	if hasEntries {
		if err := db.unlockedSave(); err != nil {
			return err
		}
	}
	if err := os.Remove(walPath); err != nil {
		return fmt.Errorf("failed to remove WAL file %s: %w", walPath, err)
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
	var corruptedLines [][]byte

	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if lineNumber == 1 {
			var header Header
			if err := json.Unmarshal(line, &header); err != nil {
				return false, fmt.Errorf("failed to parse WAL header: %w", err)
			}

			if db.Data.Lineage == "" && header.Lineage != "" {
				db.Data.Lineage = header.Lineage
			} else if db.Data.Lineage != header.Lineage {
				return false, fmt.Errorf("WAL lineage (%q) does not match state lineage (%q)", header.Lineage, db.Data.Lineage)
			}

			expectedSerial := db.Data.Serial + 1
			if header.Serial < expectedSerial {
				return false, errStaleWAL
			}
			if header.Serial > expectedSerial {
				return false, fmt.Errorf("WAL serial (%d) is ahead of expected (%d), state may be corrupted", header.Serial, expectedSerial)
			}
			db.Data.Serial = expectedSerial
		} else {
			var entry WALEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				log.Warnf(ctx, "Skipping corrupted WAL entry at %s:%d: %v", walPath, lineNumber, err)
				corruptedLines = append(corruptedLines, append([]byte(nil), line...))
				continue
			}
			if db.Data.State == nil {
				db.Data.State = make(map[string]ResourceEntry)
			}
			if entry.Value == nil {
				delete(db.Data.State, entry.Key)
				delete(db.stateIDs, entry.Key)
			} else {
				db.Data.State[entry.Key] = *entry.Value
				db.stateIDs[entry.Key] = entry.Value.ID
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	if len(corruptedLines) > 0 {
		corruptedPath := walPath + ".corrupted"
		corruptedData := bytes.Join(corruptedLines, []byte("\n"))
		if writeErr := os.WriteFile(corruptedPath, corruptedData, 0o600); writeErr != nil {
			log.Warnf(ctx, "Failed to save corrupted WAL entries to %s: %v", corruptedPath, writeErr)
		} else {
			log.Warnf(ctx, "Saved %d corrupted WAL entries to %s", len(corruptedLines), corruptedPath)
		}
	}

	return lineNumber > 1, nil
}

// Finalize replays the WAL (if open for write), captures the resulting state, and resets.
// Safe to call multiple times or on an already-finalized state.
// Returns the exported state as of the end of this operation.
func (db *DeploymentState) Finalize(ctx context.Context) (resourcestate.ExportedResourcesMap, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.Path == "" {
		return nil, nil
	}

	var err error

	if db.walFile != nil {
		closeErr := db.walFile.Close()
		if closeErr != nil {
			log.Warnf(ctx, "Error when closing .wal file, possibly corrupted state file: %s", closeErr)
		}
		db.walFile = nil
		err = db.replayWAL(ctx)
	}

	state := ExportStateFromData(db.Data)

	db.Path = ""
	db.Data = Database{}
	db.stateIDs = nil

	return state, err
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
	if err := os.MkdirAll(filepath.Dir(walPath), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	walFile, err := os.OpenFile(walPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open WAL file %s: %w", walPath, err)
	}
	db.walFile = walFile

	lineage := db.Data.Lineage
	if lineage == "" {
		lineage = uuid.New().String()
	}
	walHead := Header{
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

// ExportStateFromData extracts resource IDs and ETags from a database snapshot.
func ExportStateFromData(data Database) resourcestate.ExportedResourcesMap {
	result := make(resourcestate.ExportedResourcesMap)
	for key, entry := range data.State {
		var etag string
		// Extract etag for resources that use it for drift detection
		// (dashboards and genie_spaces). Both follow the same pattern of
		// persisting the backend-returned etag in state and comparing it
		// against the remote on the next plan via OverrideChangeDesc.
		// covered by test cases:
		//   - bundle/deploy/dashboard/detect-change
		//   - bundle/resources/genie_spaces/simple
		if (strings.Contains(key, ".dashboards.") || strings.Contains(key, ".genie_spaces.")) && len(entry.State) > 0 {
			var holder struct {
				Etag string `json:"etag"`
			}
			if err := json.Unmarshal(entry.State, &holder); err == nil {
				etag = holder.Etag
			}
		}

		result[key] = resourcestate.ResourceState{
			ID:                       entry.ID,
			ETag:                     etag,
			StateSizeBytes:           len(entry.State),
			StateCompressedSizeBytes: compressedStateSize(entry.State),
		}
	}
	return result
}

func (db *DeploymentState) ExportState(ctx context.Context) resourcestate.ExportedResourcesMap {
	return ExportStateFromData(db.Data)
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
