package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagerCanDisableIdleShutdown(t *testing.T) {
	manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
	require.True(t, manager.TryAdd("session", newProxyConnection(nil)))
	manager.Remove("session")
	select {
	case <-manager.TimedOut:
		t.Fatal("idle shutdown must stay disabled")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestSingleSessionRejectsReuseAfterManagerRemoval(t *testing.T) {
	manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
	handler := NewProxyServer(t.Context(), manager, func(ctx context.Context) *exec.Cmd {
		t.Fatal("a completed logical session must never start another process")
		return nil
	}, WithSingleSession())
	// Simulate a session that already ended and was removed from the manager.
	handler.sessionUsed = true
	for _, id := range []string{"same-session", "different-session"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ssh?id="+id, nil))
		require.Equal(t, http.StatusServiceUnavailable, response.Code)
	}
}

func TestSingleSessionFailedUpgradeDoesNotConsumeSession(t *testing.T) {
	completed := false
	manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
	handler := NewProxyServer(t.Context(), manager, func(ctx context.Context) *exec.Cmd {
		t.Fatal("a non-WebSocket request must never start a process")
		return nil
	}, WithSingleSession(), WithSessionCompletion(func() { completed = true }))
	for range 2 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ssh?id=test", nil))
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.False(t, handler.sessionUsed)
		require.False(t, completed)
		require.Zero(t, manager.Count())
	}
}

func TestZeroIdleTimeoutStillSignalsImmediately(t *testing.T) {
	manager := NewConnectionsManager(1, 0)
	select {
	case <-manager.TimedOut:
	case <-time.After(5 * time.Second):
		t.Fatal("zero duration must preserve compute's immediate idle shutdown")
	}
}
