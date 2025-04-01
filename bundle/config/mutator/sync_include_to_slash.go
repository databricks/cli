package mutator

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

// We need to keep include and exclude as Unix slash path in order for
// ignore.GitIgnore we use in libs/fileset to work correctly.
func SyncIncludeExcludeToSlash() bundle.Mutator {
	return &syncIncludeExcludeToSlash{}
}

type syncIncludeExcludeToSlash struct{}

func (m *syncIncludeExcludeToSlash) Name() string {
	return "SyncIncludeExcludeToSlash"
}

func (m *syncIncludeExcludeToSlash) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for i := range b.Config.Sync.Include {
		b.Config.Sync.Include[i] = filepath.ToSlash(b.Config.Sync.Include[i])
	}
	for i := range b.Config.Sync.Exclude {
		b.Config.Sync.Exclude[i] = filepath.ToSlash(b.Config.Sync.Exclude[i])
	}
	return nil
}
