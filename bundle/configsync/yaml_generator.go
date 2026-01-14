package configsync

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
)

// GenerateYAMLFiles generates YAML files for the given changes.
func GenerateYAMLFiles(ctx context.Context, b *bundle.Bundle, changes map[string]deployplan.Changes) ([]FileChange, error) {
	return nil, nil
}

// SaveFiles writes all file changes to disk.
func SaveFiles(ctx context.Context, b *bundle.Bundle, files []FileChange) error {
	return nil
}
