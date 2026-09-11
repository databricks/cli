package fuse

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
)

const refreshInterval = 10 * time.Minute

// TokenFunc returns the current credential, including any refresh performed by the SDK.
type TokenFunc func(context.Context) (string, error)

// KeepRegistered makes an initial attempt before returning, then retries in the background,
// even if that attempt failed. Cancellation stops refresh; entries remain until compute shutdown.
func KeepRegistered(ctx context.Context, client *Client, token TokenFunc, userID string) error {
	err := refresh(ctx, client, token, userID, false)
	go refreshUntilDone(ctx, client, token, userID)
	return err
}

func refresh(ctx context.Context, client *Client, token TokenFunc, userID string, force bool) error {
	value, err := token(ctx)
	if err != nil {
		return fmt.Errorf("failed to get FUSE credentials: %w", err)
	}
	return client.Register(ctx, value, userID, force)
}

func refreshUntilDone(ctx context.Context, client *Client, token TokenFunc, userID string) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	defer client.http.CloseIdleConnections()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pid, err := client.readPID()
			// Re-register only after credential rotation or loss of the registered ancestor.
			// Unchanged registrations unnecessarily invalidate the daemon's caches.
			force := err != nil || pid != client.registration.PID
			if err := refresh(ctx, client, token, userID, force); err != nil {
				log.Warnf(ctx, "Failed to refresh SSH filesystem credentials; retrying in %v: %v", refreshInterval, err)
			}
		}
	}
}

func registeredPID() (int, error) {
	content, err := os.ReadFile("/Workspace/.proc/self/metadata/pid")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(content)))
}
