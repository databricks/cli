package dstate

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

type WAL struct {
	file *os.File
}

type corruptedWALEntry struct {
	lineNumber int
	rawLine    string
	parseErr   error
}

type walReplayResult struct {
	recovered        bool
	stale            bool
	entriesRecovered int
	corruptedEntries []corruptedWALEntry
}

var errWALRead = errors.New("wal read error")

func walPath(statePath string) string {
	return statePath + ".wal"
}

func walCorruptedPath(statePath string) string {
	return walPath(statePath) + ".corrupted"
}

func openWAL(statePath string) (*WAL, error) {
	wp := walPath(statePath)
	f, err := os.OpenFile(wp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file %q: %w", wp, err)
	}
	return &WAL{file: f}, nil
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

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync WAL entry: %w", err)
	}

	return nil
}

func (w *WAL) close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func cleanupWAL(statePath string) error {
	err := os.Remove(walPath(statePath))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove WAL file %q: %w", walPath(statePath), err)
	}
	return nil
}

func moveWALToCorrupted(statePath string) error {
	source := walPath(statePath)
	target := walCorruptedPath(statePath)
	_ = os.Remove(target)
	if err := os.Rename(source, target); err != nil {
		return fmt.Errorf("failed to move WAL file %q to %q: %w", source, target, err)
	}
	return nil
}

func writeCorruptedWALEntries(statePath string, corrupted []corruptedWALEntry) error {
	if len(corrupted) == 0 {
		return nil
	}

	target := walCorruptedPath(statePath)
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create corrupted WAL file %q: %w", target, err)
	}
	defer f.Close()

	for _, entry := range corrupted {
		if _, err := f.WriteString(entry.rawLine + "\n"); err != nil {
			return fmt.Errorf("failed to write corrupted WAL file %q: %w", target, err)
		}
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync corrupted WAL file %q: %w", target, err)
	}

	return nil
}

func readWAL(statePath string) (*WALHeader, []WALEntry, []corruptedWALEntry, error) {
	wp := walPath(statePath)
	f, err := os.Open(wp)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	var header *WALHeader
	var entries []WALEntry
	var corrupted []corruptedWALEntry
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		lineCopy := make([]byte, len(line))
		copy(lineCopy, line)
		if header == nil {
			var h WALHeader
			if err := json.Unmarshal(lineCopy, &h); err != nil {
				return nil, nil, nil, fmt.Errorf("failed to parse WAL header: %w", err)
			}
			header = &h
			continue
		}

		var e WALEntry
		if err := json.Unmarshal(lineCopy, &e); err != nil {
			corrupted = append(corrupted, corruptedWALEntry{
				lineNumber: lineNumber,
				rawLine:    string(lineCopy),
				parseErr:   err,
			})
			continue
		}

		if e.K == "" {
			corrupted = append(corrupted, corruptedWALEntry{
				lineNumber: lineNumber,
				rawLine:    string(lineCopy),
				parseErr:   errors.New("entry has empty key"),
			})
			continue
		}

		entries = append(entries, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read WAL file: %w", err)
	}

	if header == nil {
		return nil, nil, nil, errors.New("WAL file is empty")
	}

	return header, entries, corrupted, nil
}

func replayWAL(statePath string, db *Database) (walReplayResult, error) {
	result := walReplayResult{}
	header, entries, corrupted, err := readWAL(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("%w: %v", errWALRead, err)
	}

	expectedSerial := db.Serial + 1
	if header.Serial < expectedSerial {
		result.stale = true
		return result, nil
	}

	if header.Serial > expectedSerial {
		return result, fmt.Errorf("WAL serial (%d) is ahead of expected (%d), state may be corrupted", header.Serial, expectedSerial)
	}

	if db.Lineage != "" && header.Lineage != "" && db.Lineage != header.Lineage {
		return result, fmt.Errorf("WAL lineage (%s) does not match state lineage (%s)", header.Lineage, db.Lineage)
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

	result.recovered = true
	result.entriesRecovered = len(entries)
	result.corruptedEntries = corrupted
	return result, nil
}

func recoverFromWAL(ctx context.Context, statePath string, db *Database) (bool, error) {
	replayResult, err := replayWAL(statePath, db)
	if err != nil {
		if errors.Is(err, errWALRead) {
			if moveErr := moveWALToCorrupted(statePath); moveErr != nil {
				return false, moveErr
			}
			log.Warnf(ctx, "Failed to read WAL file, moved it to %s and proceeding: %s", relativePathForLog(walCorruptedPath(statePath)), strings.TrimPrefix(err.Error(), errWALRead.Error()+": "))
			return false, nil
		}
		return false, err
	}

	if replayResult.stale {
		log.Warnf(ctx, "Deleting stale WAL (serial=%s behind current state serial=)")
		if err := cleanupWAL(statePath); err != nil {
			return false, err
		}
		return false, nil
	}

	if !replayResult.recovered {
		return false, nil
	}

	logRecoveryProgress(ctx, "Recovering state from WAL file: "+relativePathForLog(walPath(statePath)))
	walLogPath := relativePathForLog(walPath(statePath))
	for _, corrupted := range replayResult.corruptedEntries {
		log.Warnf(ctx, "Could not read state file WAL entry in %s: line %d", walLogPath, corrupted.lineNumber)
	}

	if err := writeCorruptedWALEntries(statePath, replayResult.corruptedEntries); err != nil {
		return false, err
	}
	if len(replayResult.corruptedEntries) > 0 {
		log.Warnf(ctx, "Saved corrupted WAL entries to %s", relativePathForLog(walCorruptedPath(statePath)))
	}

	logRecoveryProgress(ctx, fmt.Sprintf("Recovered %d entries from WAL file.", replayResult.entriesRecovered))
	return true, nil
}

func relativePathForLog(path string) string {
	rel, err := filepath.Rel(".", path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func logRecoveryProgress(ctx context.Context, message string) {
	defer func() {
		_ = recover()
	}()
	cmdio.LogString(ctx, message)
}
