package trajectory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Writer struct {
	file *os.File
	mu   sync.Mutex
}

func NewWriter(historyPath string) (*Writer, error) {
	if err := os.MkdirAll(filepath.Dir(historyPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}

	return &Writer{
		file: file,
	}, nil
}

func (w *Writer) WriteEntry(entry HistoryEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	if _, err := w.file.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return w.file.Sync()
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
