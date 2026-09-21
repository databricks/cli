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

const (
	refreshInterval = time.Minute
	pidProbeTimeout = 2 * time.Second
)

// TokenFunc returns the current credential, including any refresh performed by the SDK.
type TokenFunc func(context.Context) (string, error)

// KeepRegistered makes an initial attempt before returning, then retries in the background,
// even if that attempt failed. Cancellation stops refresh; entries remain until compute shutdown.
func KeepRegistered(ctx context.Context, client *Client, token TokenFunc, userID, notebookDir string) error {
	err := refresh(ctx, client, token, userID, notebookDir, false)
	go refreshUntilDone(ctx, client, token, userID, notebookDir)
	return err
}

func refresh(ctx context.Context, client *Client, token TokenFunc, userID, notebookDir string, force bool) error {
	value, err := token(ctx)
	if err != nil {
		return fmt.Errorf("failed to get FUSE credentials: %w", err)
	}
	return client.Register(ctx, value, userID, notebookDir, force)
}

func refreshUntilDone(ctx context.Context, client *Client, token TokenFunc, userID, notebookDir string) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	defer client.http.CloseIdleConnections()
	var pendingProbe chan bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if pendingProbe == nil {
				pendingProbe = make(chan bool, 1)
				go func(result chan<- bool) {
					pid, err := client.readPID()
					result <- err != nil || pid != client.registration.PID
				}(pendingProbe)
			}
			// Re-register WSFS only after credential rotation or loss of the registered ancestor.
			// Unchanged WSFS registrations unnecessarily invalidate the daemon's caches.
			force := true
			timer := time.NewTimer(pidProbeTimeout)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case force = <-pendingProbe:
				pendingProbe = nil
			case <-timer.C:
			}
			timer.Stop()
			if err := refresh(ctx, client, token, userID, notebookDir, force); err != nil {
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
