package dstate

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// WALHeader is the first entry in the WAL file, containing metadata for validation.
type WALHeader struct {
	Lineage string `json:"lineage"`
	Serial  int    `json:"serial"`
}

// WALEntry represents a single state mutation in the WAL.
// For set operations, V is populated. For delete operations, V is nil.
type WALEntry struct {
	K string         `json:"k"`
	V *ResourceEntry `json:"v,omitempty"`
}

// WAL manages the Write-Ahead Log for deployment state recovery.
type WAL struct {
	path string
	file *os.File
}

// walPath returns the WAL file path for a given state file path.
func walPath(statePath string) string {
	return statePath + ".wal"
}

// openWAL opens or creates a WAL file for writing.
func openWAL(statePath string) (*WAL, error) {
	wp := walPath(statePath)
	f, err := os.OpenFile(wp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file %q: %w", wp, err)
	}
	return &WAL{path: wp, file: f}, nil
}

// writeHeader writes the WAL header (lineage and serial) as the first entry.
func (w *WAL) writeHeader(lineage string, serial int) error {
	header := WALHeader{
		Lineage: lineage,
		Serial:  serial,
	}
	return w.writeJSON(header)
}

// writeEntry appends a state mutation entry to the WAL.
func (w *WAL) writeEntry(key string, entry *ResourceEntry) error {
	walEntry := WALEntry{
		K: key,
		V: entry,
	}
	return w.writeJSON(walEntry)
}

// writeJSON marshals and writes a JSON object as a single line, then syncs to disk.
func (w *WAL) writeJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal WAL entry: %w", err)
	}
	data = append(data, '\n')

	_, err = w.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write WAL entry: %w", err)
	}

	err = w.file.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync WAL file: %w", err)
	}

	return nil
}

// close closes the WAL file handle.
func (w *WAL) close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// truncate deletes the WAL file after successful finalization.
func (w *WAL) truncate() error {
	if w.file != nil {
		w.file.Close()
		w.file = nil
	}
	err := os.Remove(w.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove WAL file %q: %w", w.path, err)
	}
	return nil
}

// readWAL reads and parses an existing WAL file for recovery.
// Returns the header and entries, or an error if the WAL is invalid.
func readWAL(statePath string) (*WALHeader, []WALEntry, error) {
	wp := walPath(statePath)
	f, err := os.Open(wp)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var header *WALHeader
	var entries []WALEntry
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		if header == nil {
			// First line must be the header
			var h WALHeader
			if err := json.Unmarshal(line, &h); err != nil {
				return nil, nil, fmt.Errorf("WAL line %d: failed to parse header: %w", lineNum, err)
			}
			header = &h
		} else {
			// Subsequent lines are entries
			var e WALEntry
			if err := json.Unmarshal(line, &e); err != nil {
				// Skip corrupted lines silently - this is expected for partial writes
				continue
			}
			if e.K == "" {
				// Skip entries with empty keys
				continue
			}
			entries = append(entries, e)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read WAL file: %w", err)
	}

	if header == nil {
		return nil, nil, errors.New("WAL file is empty or missing header")
	}

	return header, entries, nil
}

// recoverFromWAL attempts to recover state from an existing WAL file.
// It validates the WAL against the current state and replays valid entries.
// Returns true if recovery was performed, false if no recovery needed.
func recoverFromWAL(statePath string, db *Database) (bool, error) {
	wp := walPath(statePath)

	// Check if WAL exists
	if _, err := os.Stat(wp); os.IsNotExist(err) {
		return false, nil
	}

	header, entries, err := readWAL(statePath)
	if err != nil {
		// If we can't read the WAL at all, delete it and proceed
		os.Remove(wp)
		return false, nil
	}

	// Validate WAL serial against state serial
	expectedSerial := db.Serial + 1
	if header.Serial < expectedSerial {
		// Stale WAL - delete and proceed without recovery
		os.Remove(wp)
		return false, nil
	}

	if header.Serial > expectedSerial {
		// WAL is ahead of state - this indicates corruption
		return false, fmt.Errorf("WAL serial (%d) is ahead of expected (%d), state may be corrupted", header.Serial, expectedSerial)
	}

	// Validate lineage if both exist
	if db.Lineage != "" && header.Lineage != "" && db.Lineage != header.Lineage {
		return false, fmt.Errorf("WAL lineage (%s) does not match state lineage (%s)", header.Lineage, db.Lineage)
	}

	// Adopt lineage from WAL if state doesn't have one
	if db.Lineage == "" && header.Lineage != "" {
		db.Lineage = header.Lineage
	}

	// Initialize state map if needed
	if db.State == nil {
		db.State = make(map[string]ResourceEntry)
	}

	// Replay entries
	for _, entry := range entries {
		if entry.V != nil {
			// Set operation
			db.State[entry.K] = *entry.V
		} else {
			// Delete operation
			delete(db.State, entry.K)
		}
	}

	return true, nil
}
