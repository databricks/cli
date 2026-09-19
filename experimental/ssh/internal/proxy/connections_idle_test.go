package proxy_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionsManagerExtendIdleTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		connections := proxy.NewConnectionsManager(1, time.Minute)
		connections.ExtendIdleTimeout(time.Second)
		time.Sleep(time.Second)
		assert.True(t, <-connections.TimedOut)
		require.True(t, connections.TryAdd("connected", nil))
		connections.ExtendIdleTimeout(time.Second)
		time.Sleep(time.Minute)
		synctest.Wait()
		select {
		case <-connections.TimedOut:
			t.Fatal("extended timeout fired while a client was connected")
		default:
		}
	})
}
