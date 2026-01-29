package configsync

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
)

// FileChange represents a change to a bundle configuration file
type FileChange struct {
	Path            string `json:"path"`
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

// DiffOutput represents the complete output of the config-remote-sync command
type DiffOutput struct {
	Files   []FileChange                  `json:"files"`
	Changes map[string]deployplan.Changes `json:"changes"`
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
