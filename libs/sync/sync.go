package sync

import (
	"context"
	"time"

	"github.com/databricks/bricks/libs/sync/repofiles"
	"github.com/databricks/databricks-sdk-go"
)

type Sync struct {
	LocalPath  string
	RemotePath string

	PersistSnapshot bool

	PollInterval time.Duration
}

// RunWatchdog kicks off a polling loop to monitor local changes and synchronize
// them to the remote workspace path.
func (s *Sync) RunWatchdog(ctx context.Context, wsc *databricks.WorkspaceClient) error {
	repoFiles := repofiles.Create(s.RemotePath, s.LocalPath, wsc)
	syncCallback := syncCallback(ctx, repoFiles)
	return spawnWatchdog(ctx, s.PollInterval, syncCallback, s.RemotePath, s.PersistSnapshot)
}
