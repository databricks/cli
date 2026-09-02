package configsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// FileChange represents a change to a bundle configuration file
type FileChange struct {
	Path            string `json:"path"`
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

// SyncStatus is the outcome the JSON caller (DABs in the Workspace) branches on.
type SyncStatus string

const (
	StatusSuccess SyncStatus = "success"
	// StatusSkipped: nothing to sync against (state missing, or no selector
	// matched a deployed resource). Not an error for the caller.
	StatusSkipped SyncStatus = "skipped"
	StatusFailed  SyncStatus = "failed"
)

// DiffOutput represents the complete output of the config-remote-sync command
type DiffOutput struct {
	Status  SyncStatus   `json:"status"`
	Error   string       `json:"error,omitempty"`
	Files   []FileChange `json:"files"`
	Changes Changes      `json:"changes"`
}

// WriteResult renders the result and records the error payload on stats for
// telemetry. In JSON mode the process never fails: the outcome is carried in
// DiffOutput.Status so the caller branches on it, not the exit code. Text mode
// returns the error so the CLI still fails loudly for humans.
func WriteResult(out io.Writer, jsonOutput bool, stats *Stats, files []FileChange, changes Changes, err error) error {
	status := StatusSuccess
	if err != nil {
		if stats.ErrorCategory == "" {
			stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryBundleLoadFailed
		}
		stats.ErrorMessage = telemetry.ScrubErrorMessage(err.Error())
		// Missing state or a selector that matched nothing: nothing to sync, not a failure.
		if errors.Is(err, ErrStateSnapshotNotFound) || errors.Is(err, ErrNoMatchingSelector) {
			status = StatusSkipped
		} else {
			status = StatusFailed
		}
	}

	if !jsonOutput {
		if err != nil {
			return err
		}
		_, _ = io.WriteString(out, FormatTextOutput(changes))
		_, _ = out.Write([]byte{'\n'})
		return nil
	}

	output := DiffOutput{Status: status, Files: files, Changes: changes}
	if output.Changes == nil {
		// Serialize as {} rather than null so a consumer can iterate it unconditionally.
		output.Changes = Changes{}
	}
	if err != nil {
		output.Error = err.Error()
	}
	result, marshalErr := json.MarshalIndent(output, "", "  ")
	if marshalErr != nil {
		// Cannot produce JSON: fail rather than emit a partial contract.
		stats.ErrorCategory = protos.BundleConfigRemoteSyncErrorCategoryOutputFailed
		stats.ErrorMessage = telemetry.ScrubErrorMessage(marshalErr.Error())
		return fmt.Errorf("failed to marshal output: %w", marshalErr)
	}
	_, _ = out.Write(result)
	_, _ = out.Write([]byte{'\n'})
	return nil
}

// SaveFiles writes all file changes to disk.
func SaveFiles(ctx context.Context, b *bundle.Bundle, files []FileChange) error {
	for _, file := range files {
		err := os.MkdirAll(filepath.Dir(file.Path), 0o755)
		if err != nil {
			return err
		}

		err = os.WriteFile(file.Path, []byte(file.ModifiedContent), 0o644)
		if err != nil {
			return err
		}
	}
	return nil
}
