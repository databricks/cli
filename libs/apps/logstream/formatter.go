package logstream

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
)

// wsEntry represents a structured log entry from the websocket stream.
type wsEntry struct {
	Source    string  `json:"source"`
	Timestamp float64 `json:"timestamp"`
	Message   string  `json:"message"`
}

// parseLogEntry parses a raw log entry from the websocket stream.
func parseLogEntry(raw []byte) (*wsEntry, error) {
	var entry wsEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// logFormatter formats log entries for output.
type logFormatter struct {
	colorize     bool
	outputFormat flags.Output
}

// newLogFormatter creates a new log formatter.
func newLogFormatter(colorize bool, outputFormat flags.Output) *logFormatter {
	return &logFormatter{colorize: colorize, outputFormat: outputFormat}
}

// FormatEntry formats a structured log entry for output.
func (f *logFormatter) FormatEntry(ctx context.Context, entry *wsEntry) string {
	if f.outputFormat == flags.OutputJSON {
		return f.formatEntryJSON(entry)
	}
	return f.formatEntryText(ctx, entry)
}

// formatEntryText formats a structured log entry as human-readable text.
func (f *logFormatter) formatEntryText(ctx context.Context, entry *wsEntry) string {
	timestamp := formatTimestamp(entry.Timestamp)
	source := strings.ToUpper(entry.Source)
	message := strings.TrimRight(entry.Message, "\r\n")

	if f.colorize {
		timestamp = cmdio.HiBlack(ctx, timestamp)
		source = cmdio.HiBlue(ctx, source)
	}

	return fmt.Sprintf("%s [%s] %s", timestamp, source, message)
}

// formatEntryJSON formats a structured log entry as JSON (NDJSON line).
// On marshal failure it falls back to the plain text path; that fallback is
// uncolored because we have no ctx at that point and JSON output is never
// piped to a TTY-colored renderer anyway.
func (f *logFormatter) formatEntryJSON(entry *wsEntry) string {
	normalized := wsEntry{
		Source:    strings.ToUpper(entry.Source),
		Timestamp: entry.Timestamp,
		Message:   strings.TrimRight(entry.Message, "\r\n"),
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Sprintf("%s [%s] %s",
			formatTimestamp(entry.Timestamp),
			strings.ToUpper(entry.Source),
			strings.TrimRight(entry.Message, "\r\n"),
		)
	}
	return string(data)
}

// FormatPlain formats a plain text message by trimming trailing newlines.
func (f *logFormatter) FormatPlain(raw []byte) string {
	return strings.TrimRight(string(raw), "\r\n")
}

// formatTimestamp formats a timestamp as a string.
func formatTimestamp(ts float64) string {
	if ts <= 0 {
		return "----------"
	}
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()
	return t.Format(time.RFC3339)
}
