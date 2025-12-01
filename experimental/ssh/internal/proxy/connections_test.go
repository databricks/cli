package proxy

import (
	"fmt"
	"sync"
	"testing"
	"time"

	// TODO: re-enable synctests after we update to Go 1.25
	// "testing/synctest"

	"github.com/stretchr/testify/assert"
)

func TestConnectionsManager_TryAdd_Success(t *testing.T) {
	cm := NewConnectionsManager(2, time.Hour)
	conn1 := &proxyConnection{}
	conn2 := &proxyConnection{}

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
	cm := NewConnectionsManager(1, time.Hour)
	conn1 := &proxyConnection{}
	conn2 := &proxyConnection{}

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
	cm := NewConnectionsManager(3, time.Hour)
	conn := &proxyConnection{}

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

func TestConnectionsManager_ThreadSafety(t *testing.T) {
	const numGoroutines = 10
	const numOperationsPerGoroutine = 100
	cm := NewConnectionsManager(numGoroutines*numOperationsPerGoroutine, time.Hour)

	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Go(func() {
			for j := range numOperationsPerGoroutine {
				cm.TryAdd(fmt.Sprintf("conn-%d-%d", i, j), &proxyConnection{})
			}
		})
	}

	wg.Wait()

	// Verify we can get the count without race conditions
	countAfterAdds := cm.Count()
	assert.Equal(t, numGoroutines*numOperationsPerGoroutine, countAfterAdds)

	// Now do concurrent gets, removes, and counts
	for i := range numGoroutines {
		wg.Go(func() {
			for j := range numOperationsPerGoroutine {
				cm.Get(fmt.Sprintf("conn-%d-%d", i, j))
			}
		})
	}

	for i := range numGoroutines {
		wg.Go(func() {
			for j := range numOperationsPerGoroutine {
				cm.Remove(fmt.Sprintf("conn-%d-%d", i, j))
			}
		})
	}

	for range numGoroutines {
		wg.Go(func() {
			for range numOperationsPerGoroutine {
				cm.Count()
			}
		})
	}

	wg.Wait()

	// Test should complete without race conditions or panics
	// Final count should be 0 since we remove everything we add
	finalCount := cm.Count()
	assert.Equal(t, 0, finalCount)
}

// TODO: re-enable synctests after we update to Go 1.25

// func TestConnectionsManager_ShutdownTimer_TriggersOnEmptyConnections(t *testing.T) {
// 	synctest.Test(t, func(t *testing.T) {
// 		cm := NewConnectionsManager(3, time.Second)
// 		timedOut := false
// 		go func() {
// 			select {
// 			case <-cm.TimedOut:
// 				timedOut = true
// 			case <-time.After(time.Hour):
// 			}
// 		}()
// 		time.Sleep(time.Hour)
// 		synctest.Wait()
// 		assert.True(t, timedOut, "Expected timeout signal but didn't receive one")
// 	})
// }

// func TestConnectionsManager_ShutdownTimer_CancelledWhenConnectionAdded(t *testing.T) {
// 	synctest.Test(t, func(t *testing.T) {
// 		cm := NewConnectionsManager(3, time.Second)
// 		timedOut := false
// 		go func() {
// 			select {
// 			case <-cm.TimedOut:
// 				timedOut = true
// 			case <-time.After(time.Hour):
// 			}
// 		}()

// 		// Add connection to cancel shutdown timer
// 		conn := &proxyConnection{}
// 		result := cm.TryAdd("test-id", conn)
// 		require.True(t, result)

// 		time.Sleep(time.Hour)
// 		synctest.Wait()
// 		assert.False(t, timedOut, "Unexpected timeout signal while connection exists")
// 	})
// }

// func TestConnectionsManager_ShutdownTimer_RestartsWhenLastConnectionRemoved(t *testing.T) {
// 	synctest.Test(t, func(t *testing.T) {
// 		cm := NewConnectionsManager(3, time.Second)
// 		conn := &proxyConnection{}
// 		timedOut := false
// 		go func() {
// 			select {
// 			case <-cm.TimedOut:
// 				timedOut = true
// 			case <-time.After(time.Hour):
// 			}
// 		}()

// 		// Add connection
// 		result := cm.TryAdd("test-id", conn)
// 		require.True(t, result)

// 		// Wait a bit to ensure timer would have triggered if not cancelled
// 		time.Sleep(time.Hour)
// 		synctest.Wait()
// 		assert.False(t, timedOut, "Unexpected timeout signal while connection exists")

// 		// Setup new goroutine to listen for timeout signal
// 		timedOut = false
// 		go func() {
// 			select {
// 			case <-cm.TimedOut:
// 				timedOut = true
// 			case <-time.After(time.Hour):
// 			}
// 		}()
// 		// Remove connection - should restart shutdown timer
// 		cm.Remove("test-id")
// 		time.Sleep(time.Hour)
// 		synctest.Wait()
// 		assert.True(t, timedOut, "Expected timeout signal after last connection removed but didn't receive one")
// 	})
// }

// func TestConnectionsManager_ShutdownTimer_NoRestartWhenConnectionsRemain(t *testing.T) {
// 	synctest.Test(t, func(t *testing.T) {
// 		cm := NewConnectionsManager(3, time.Second)
// 		timedOut := false
// 		go func() {
// 			select {
// 			case <-cm.TimedOut:
// 				timedOut = true
// 			case <-time.After(time.Hour):
// 			}
// 		}()
// 		conn1 := &proxyConnection{}
// 		conn2 := &proxyConnection{}

// 		// Add two connections
// 		result := cm.TryAdd("conn1", conn1)
// 		require.True(t, result)
// 		result = cm.TryAdd("conn2", conn2)
// 		require.True(t, result)

// 		// Remove one connection - timer should not restart since connections remain
// 		cm.Remove("conn1")
// 		assert.Equal(t, 1, cm.Count())

// 		time.Sleep(time.Hour)
// 		synctest.Wait()
// 		assert.False(t, timedOut, "Unexpected timeout signal while connections still exist")
// 	})
// }
