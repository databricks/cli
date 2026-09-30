package server_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const detachedProcessDirectory = "400"

func TestWaitForIdleShutdown(t *testing.T) {
	for _, test := range []struct {
		name         string
		keepDetached bool
		hasDetached  bool
		unreadable   bool
		wantHold     bool
	}{
		{name: "flag off with detached work", hasDetached: true},
		{name: "flag off without detached work"},
		{name: "flag on without detached work", keepDetached: true},
		{name: "flag on with detached work", keepDetached: true, hasDetached: true, wantHold: true},
		{name: "flag on with unreadable process tree", keepDetached: true, unreadable: true, wantHold: true},
		{name: "flag off does not need process tree", unreadable: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				root := server.ProcWithDetachedWork(t)
				if !test.hasDetached {
					require.NoError(t, os.RemoveAll(filepath.Join(root, detachedProcessDirectory)))
				}
				if test.unreadable {
					root = t.TempDir()
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				connections := proxy.NewConnectionsManager(1, time.Second)
				shutdown := make(chan error, 1)
				go func() {
					shutdown <- server.WaitForIdleShutdown(ctx, connections, test.keepDetached, root, server.TestServerPid)
				}()
				time.Sleep(time.Second + 2*server.DetachedProcessCheckInterval)
				synctest.Wait()
				assert.Equal(t, !test.wantHold, len(shutdown) > 0)
				require.True(t, connections.TryAdd("cleanup", nil))
				cancel()
				synctest.Wait()
				require.Len(t, shutdown, 1)
				shutdownErr := <-shutdown
				if test.wantHold {
					assert.ErrorIs(t, shutdownErr, context.Canceled)
				} else {
					assert.NoError(t, shutdownErr)
				}
			})
		})
	}
}

func TestWaitForIdleShutdownWhenDetachedWorkFinishes(t *testing.T) {
	for _, shutdownDelay := range []time.Duration{0, time.Second} {
		t.Run(shutdownDelay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				root := server.ProcWithDetachedWork(t)
				connections := proxy.NewConnectionsManager(1, shutdownDelay)
				shutdown := make(chan error, 1)
				go func() {
					shutdown <- server.WaitForIdleShutdown(t.Context(), connections, true, root, server.TestServerPid)
				}()
				time.Sleep(shutdownDelay)
				synctest.Wait()
				require.Empty(t, shutdown)
				require.NoError(t, os.RemoveAll(filepath.Join(root, detachedProcessDirectory)))
				time.Sleep(server.DetachedProcessCheckInterval)
				synctest.Wait()
				require.Len(t, shutdown, 1)
				assert.NoError(t, <-shutdown)
			})
		})
	}
}

func TestWaitForIdleShutdownReconnect(t *testing.T) {
	for _, connectedFor := range []time.Duration{time.Second, time.Minute} {
		t.Run(connectedFor.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				root := server.ProcWithDetachedWork(t)
				shutdownDelay := time.Minute
				connections := proxy.NewConnectionsManager(1, shutdownDelay)
				shutdown := make(chan error, 1)
				go func() {
					shutdown <- server.WaitForIdleShutdown(t.Context(), connections, true, root, server.TestServerPid)
				}()
				time.Sleep(shutdownDelay)
				synctest.Wait()
				require.Empty(t, shutdown)
				require.True(t, connections.TryAdd("reconnected", nil))
				require.NoError(t, os.RemoveAll(filepath.Join(root, detachedProcessDirectory)))
				time.Sleep(connectedFor)
				synctest.Wait()
				require.Empty(t, shutdown)
				connections.Remove("reconnected")
				time.Sleep(shutdownDelay - time.Second)
				synctest.Wait()
				require.Empty(t, shutdown)
				time.Sleep(time.Second)
				synctest.Wait()
				require.Len(t, shutdown, 1)
				assert.NoError(t, <-shutdown)
			})
		})
	}
}
