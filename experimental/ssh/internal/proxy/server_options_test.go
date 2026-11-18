package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerCanDisableResumeAndHandover(t *testing.T) {
	manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
	defer manager.Close()
	handler := NewProxyServer(t.Context(), manager, func(ctx context.Context) *exec.Cmd {
		t.Error("rejected requests must not start an SSH process")
		return exec.CommandContext(ctx, "unused")
	}, WithoutSessionResume(), WithoutHandover())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/ssh?id=test&delivered=0&resume_version=%d", ResumeProtocolVersion), nil))
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "resume is not supported")
	require.True(t, manager.TryAdd("test", newProxyConnection(nil)))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ssh?id=test", nil))
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "handover is not supported")
}

func TestManagerCanDisableIdleShutdown(t *testing.T) {
	manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
	defer manager.Close()
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
	defer manager.Close()
	handler := NewProxyServer(t.Context(), manager, func(ctx context.Context) *exec.Cmd {
		t.Fatal("a completed logical session must never start another process")
		return nil
	}, WithSingleSession())
	// Reproduce the gap after Remove and before the outer sandbox handler returns.
	handler.sessionUsed = true
	for _, id := range []string{"same-session", "different-session"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ssh?id="+id, nil))
		require.Equal(t, http.StatusServiceUnavailable, response.Code)
	}
}

func TestSingleSessionFailedUpgradeDoesNotConsumeSession(t *testing.T) {
	manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
	defer manager.Close()
	handler := NewProxyServer(t.Context(), manager, func(ctx context.Context) *exec.Cmd {
		t.Fatal("a non-WebSocket request must never start a process")
		return nil
	}, WithSingleSession())
	for range 2 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/ssh?id=test", nil))
		require.Equal(t, http.StatusBadRequest, response.Code)
		require.False(t, handler.sessionUsed)
	}
}

func TestZeroIdleTimeoutStillSignalsImmediately(t *testing.T) {
	manager := NewConnectionsManager(1, 0)
	defer manager.Close()
	select {
	case <-manager.TimedOut:
	case <-time.After(5 * time.Second):
		t.Fatal("zero duration must preserve compute's immediate idle shutdown")
	}
}
