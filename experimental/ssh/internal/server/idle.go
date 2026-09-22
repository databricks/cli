package server

import (
	"context"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/libs/log"
)

const detachedProcessCheckInterval = 15 * time.Second

func waitForIdleShutdown(ctx context.Context, connections *proxy.ConnectionsManager, keepDetached bool, root string, selfPid int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-connections.TimedOut:
		}
		if connections.Count() > 0 {
			continue
		}
		if !keepDetached {
			return nil
		}
		pids, err := detachedDescendants(root, selfPid)
		if err == nil && len(pids) == 0 {
			return nil
		}
		if err != nil {
			log.Warnf(ctx, "Cannot check detached processes; postponing SSH idle shutdown: %v", err)
		} else {
			log.Infof(ctx, "Keeping SSH server running for %d detached process(es): %s", len(pids), formatPids(pids))
		}
		connections.ExtendIdleTimeout(detachedProcessCheckInterval)
	}
}
