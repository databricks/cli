package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestSingleSessionReportsStartupFailureAndAllowsRetry(t *testing.T) {
	for _, tc := range []struct{ stage, errorText string }{
		{"stdin", "failed to get stdin pipe"},
		{"stdout", "failed to get stdout pipe"},
		{"start", "failed to start SSHD process"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			manager := NewConnectionsManager(1, 0, WithoutIdleShutdown())
			completed := make(chan error, 1)
			handler := NewProxyServer(t.Context(), manager, func(ctx context.Context) *exec.Cmd {
				cmd := exec.CommandContext(ctx, "missing-sshd-executable")
				switch tc.stage {
				case "stdin":
					cmd.Stdin = strings.NewReader("")
				case "stdout":
					cmd.Stdout = io.Discard
				}
				return cmd
			}, WithSingleSession(), WithSessionCompletion(func(err error) { completed <- err }))
			handled := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, r)
				handled <- struct{}{}
			}))
			defer server.Close()
			for range 2 {
				conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), "ws"+server.URL[4:]+"/ssh?id=retry", nil)
				if response != nil {
					response.Body.Close()
				}
				require.NoError(t, err, "startup failure must not consume the single session")
				require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
				_, _, err = conn.ReadMessage()
				conn.Close()
				require.Error(t, err)
				select {
				case <-handled:
				case <-time.After(5 * time.Second):
					t.Fatal("failed startup did not release its handler")
				}
				require.ErrorContains(t, <-completed, tc.errorText)
				require.False(t, handler.sessionUsed)
				require.Zero(t, manager.Count())
			}
		})
	}
}

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
	}, WithSingleSession(), WithSessionCompletion(func(error) { completed = true }))
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
