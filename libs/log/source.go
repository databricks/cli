package log

import (
	"log/slog"
	"path/filepath"
)

// ReplaceSourceAttr rewrites the source attribute to include only the file's basename.
func ReplaceSourceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key != slog.SourceKey {
		return a
	}

	a.Value = slog.StringValue(filepath.Base(a.Value.String()))
	return a
}
