package server

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionsManager_TryAdd_Success(t *testing.T) {
	cm := newConnectionsManager(2, time.Hour)
	conn1 := &proxy.ProxyConnection{}
	conn2 := &proxy.ProxyConnection{}

	// Should succeed for first connection
	result := cm.TryAdd("conn1", conn1)
	assert.True(t, result)
	assert.Equal(t, 1, cm.Count())

	// Should succeed for second connection
	result = cm.TryAdd("conn2", conn2)
	assert.True(t, result)
	assert.Equal(t, 2, cm.Count())

	// Verify connections can be retrieved
	retrieved1, exists1 := cm.Get("conn1")
	assert.True(t, exists1)
	assert.Equal(t, conn1, retrieved1)

	retrieved2, exists2 := cm.Get("conn2")
	assert.True(t, exists2)
	assert.Equal(t, conn2, retrieved2)
}

func TestConnectionsManager_TryAdd_MaxClientsReached(t *testing.T) {
	cm := newConnectionsManager(1, time.Hour)
	conn1 := &proxy.ProxyConnection{}
	conn2 := &proxy.ProxyConnection{}

	// First connection should succeed
	result := cm.TryAdd("conn1", conn1)
	assert.True(t, result)
	assert.Equal(t, 1, cm.Count())

	// Second connection should fail due to max clients limit
	result = cm.TryAdd("conn2", conn2)
	assert.False(t, result)
	assert.Equal(t, 1, cm.Count())

	// Only first connection should exist
	_, exists1 := cm.Get("conn1")
	assert.True(t, exists1)

	_, exists2 := cm.Get("conn2")
	assert.False(t, exists2)
}

func TestConnectionsManager_Remove(t *testing.T) {
	cm := newConnectionsManager(3, time.Hour)
	conn := &proxy.ProxyConnection{}

	// Add connection
	result := cm.TryAdd("test-id", conn)
	assert.True(t, result)
	assert.Equal(t, 1, cm.Count())

	// Remove connection
	cm.Remove("test-id")
	assert.Equal(t, 0, cm.Count())

	// Connection should no longer exist
	_, exists := cm.Get("test-id")
	assert.False(t, exists)

	// Remove non-existent connection should not panic
	cm.Remove("non-existent")
	assert.Equal(t, 0, cm.Count())
}

func TestConnectionsManager_ShutdownTimer_TriggersOnEmptyConnections(t *testing.T) {
	shutdownDelay := 10 * time.Millisecond
	cm := newConnectionsManager(3, shutdownDelay)

	// Should timeout quickly since no connections are added
	select {
	case <-cm.TimedOut:
		// Expected - timer should trigger
	case <-time.After(shutdownDelay * 10): // Bigger timeout to avoid flakiness when system is busy
		t.Fatal("Expected timeout signal but didn't receive one")
	}
}

func TestConnectionsManager_ShutdownTimer_CancelledWhenConnectionAdded(t *testing.T) {
	shutdownDelay := 10 * time.Millisecond
	cm := newConnectionsManager(3, shutdownDelay)
	conn := &proxy.ProxyConnection{}

	// Add connection to cancel shutdown timer
	result := cm.TryAdd("test-id", conn)
	require.True(t, result)

	// Should not timeout while connection exists
	select {
	case <-cm.TimedOut:
		t.Fatal("Unexpected timeout signal while connection exists")
	case <-time.After(shutdownDelay * 2):
		// Expected - no timeout should occur
	}
}

func TestConnectionsManager_ShutdownTimer_RestartsWhenLastConnectionRemoved(t *testing.T) {
	shutdownDelay := 10 * time.Millisecond
	cm := newConnectionsManager(3, shutdownDelay)
	conn := &proxy.ProxyConnection{}

	// Add connection
	result := cm.TryAdd("test-id", conn)
	require.True(t, result)

	// Wait a bit to ensure timer would have triggered if not cancelled
	time.Sleep(shutdownDelay * 2)

	// Remove connection - should restart shutdown timer
	cm.Remove("test-id")

	// Should timeout after the delay
	select {
	case <-cm.TimedOut:
		// Expected - timer should trigger after last connection removed
	case <-time.After(shutdownDelay * 10): // Bigger timeout to avoid flakiness when system is busy
		t.Fatal("Expected timeout signal after removing last connection")
	}
}

func TestConnectionsManager_ShutdownTimer_NoRestartWhenConnectionsRemain(t *testing.T) {
	shutdownDelay := 10 * time.Millisecond
	cm := newConnectionsManager(3, shutdownDelay)
	conn1 := &proxy.ProxyConnection{}
	conn2 := &proxy.ProxyConnection{}

	// Add two connections
	result := cm.TryAdd("conn1", conn1)
	require.True(t, result)
	result = cm.TryAdd("conn2", conn2)
	require.True(t, result)

	// Remove one connection - timer should not restart since connections remain
	cm.Remove("conn1")
	assert.Equal(t, 1, cm.Count())

	// Should not timeout while connection still exists
	select {
	case <-cm.TimedOut:
		t.Fatal("Unexpected timeout signal while connections remain")
	case <-time.After(shutdownDelay * 2):
		// Expected - no timeout should occur
	}
}

func TestConnectionsManager_ThreadSafety(t *testing.T) {
	cm := newConnectionsManager(100, time.Hour)
	const numGoroutines = 10
	const numOperationsPerGoroutine = 100

	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := range numOperationsPerGoroutine {
				cm.TryAdd(fmt.Sprintf("conn-%d-%d", routineID, j), &proxy.ProxyConnection{})
			}
		}(i)
	}

	wg.Wait()

	// Verify we can get the count without race conditions
	countAfterAdds := cm.Count()
	assert.LessOrEqual(t, countAfterAdds, numGoroutines*numOperationsPerGoroutine)

	// Now do concurrent gets, removes, and counts
	for i := range numGoroutines {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := range numOperationsPerGoroutine {
				cm.Get(fmt.Sprintf("conn-%d-%d", routineID, j))
			}
		}(i)
	}

	for i := range numGoroutines {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := range numOperationsPerGoroutine {
				cm.Remove(fmt.Sprintf("conn-%d-%d", routineID, j))
			}
		}(i)
	}

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numOperationsPerGoroutine {
				cm.Count()
			}
		}()
	}

	wg.Wait()

	// Test should complete without race conditions or panics
	// Final count should be 0 since we remove everything we add
	finalCount := cm.Count()
	assert.Equal(t, 0, finalCount)
}
