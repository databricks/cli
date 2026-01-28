package dstate

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/log"
)

type WALHeader struct {
	Lineage string `json:"lineage"`
	Serial  int    `json:"serial"`
}

type WALEntry struct {
	K string         `json:"k"`
	V *ResourceEntry `json:"v,omitempty"` // nil means delete
}

type WAL struct {
	path string
	file *os.File
}

func walPath(statePath string) string {
	return statePath + ".wal"
}

func openWAL(statePath string) (*WAL, error) {
	wp := walPath(statePath)
	f, err := os.OpenFile(wp, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file %q: %w", wp, err)
	}
	return &WAL{path: wp, file: f}, nil
}

func (w *WAL) writeHeader(lineage string, serial int) error {
	header := WALHeader{
		Lineage: lineage,
		Serial:  serial,
	}
	return w.writeJSON(header)
}

func (w *WAL) writeEntry(key string, entry *ResourceEntry) error {
	walEntry := WALEntry{
		K: key,
		V: entry,
	}
	return w.writeJSON(walEntry)
}

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

	return nil
}

func (w *WAL) close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

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

func readWAL(ctx context.Context, statePath string) (*WALHeader, []WALEntry, error) {
	wp := walPath(statePath)
	f, err := os.Open(wp)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines [][]byte
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)
		lines = append(lines, lineCopy)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read WAL file: %w", err)
	}

	if len(lines) == 0 {
		return nil, nil, errors.New("WAL file is empty")
	}

	var header WALHeader
	if err := json.Unmarshal(lines[0], &header); err != nil {
		return nil, nil, fmt.Errorf("failed to parse WAL header: %w", err)
	}

	var entries []WALEntry
	for i := 1; i < len(lines); i++ {
		lineNum := i + 1
		isLastLine := i == len(lines)-1

		var e WALEntry
		if err := json.Unmarshal(lines[i], &e); err != nil {
			if isLastLine {
				log.Debugf(ctx, "WAL line %d: skipping corrupted last entry: %v", lineNum, err)
				continue
			}
			return nil, nil, fmt.Errorf("WAL line %d: corrupted entry in middle of WAL: %w", lineNum, err)
		}

		if e.K == "" {
			if isLastLine {
				log.Debugf(ctx, "WAL line %d: skipping last entry with empty key", lineNum)
				continue
			}
			return nil, nil, fmt.Errorf("WAL line %d: entry with empty key in middle of WAL", lineNum)
		}

		entries = append(entries, e)
	}

	return &header, entries, nil
}

func recoverFromWAL(ctx context.Context, statePath string, db *Database) (bool, error) {
	wp := walPath(statePath)

	if _, err := os.Stat(wp); os.IsNotExist(err) {
		return false, nil
	}

	header, entries, err := readWAL(ctx, statePath)
	if err != nil {
		log.Warnf(ctx, "Failed to read WAL file, deleting and proceeding: %v", err)
		os.Remove(wp)
		return false, nil
	}

	expectedSerial := db.Serial + 1
	if header.Serial < expectedSerial {
		log.Debugf(ctx, "Deleting stale WAL (serial %d < expected %d)", header.Serial, expectedSerial)
		os.Remove(wp)
		return false, nil
	}

	if header.Serial > expectedSerial {
		return false, fmt.Errorf("WAL serial (%d) is ahead of expected (%d), state may be corrupted", header.Serial, expectedSerial)
	}

	if db.Lineage != "" && header.Lineage != "" && db.Lineage != header.Lineage {
		return false, fmt.Errorf("WAL lineage (%s) does not match state lineage (%s)", header.Lineage, db.Lineage)
	}

	if db.Lineage == "" && header.Lineage != "" {
		db.Lineage = header.Lineage
	}

	if db.State == nil {
		db.State = make(map[string]ResourceEntry)
	}

	for _, entry := range entries {
		if entry.V != nil {
			db.State[entry.K] = *entry.V
		} else {
			delete(db.State, entry.K)
		}
	}

	return true, nil
}
