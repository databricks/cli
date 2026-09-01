package fuse

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/log"
)

// RefreshInterval is how often the registration is renewed. The token handed to
// the daemons expires, and they do not refresh it themselves: WSFS keeps using
// whatever it was last given, so a session outliving the token's lifetime starts
// failing workspace calls even though its ancestry still resolves. The runtime's
// worker side re-registers on the same 10 minute cadence; the driver side has no
// refresh at all today, which is why the tunnel has to run its own.
const RefreshInterval = 10 * time.Minute

// TokenFunc returns the workspace token to register. It is called once per
// refresh rather than once at startup so that a rotating credential (OAuth, or a
// short-lived PAT) reaches the daemons before the previous one expires.
type TokenFunc func(ctx context.Context) (string, error)

// KeepRegistered registers r with both FUSE daemons and renews the registration
// every RefreshInterval until ctx is cancelled, at which point it revokes.
//
// The first registration is synchronous, so the caller learns whether the daemons
// accepted it before going on to touch /Workspace. Failures after that are logged
// and retried on the next tick.
func KeepRegistered(ctx context.Context, client *Client, r Registration, token TokenFunc, userID string) error {
	if err := register(ctx, client, r, token, userID); err != nil {
		return err
	}
	log.Infof(ctx, "Registered %s with the FUSE daemons on %s as %s", r, client.Host(), CommandOrigin)

	go refreshUntilDone(ctx, client, r, token, userID)
	return nil
}

func register(ctx context.Context, client *Client, r Registration, token TokenFunc, userID string) error {
	value, err := token(ctx)
	if err != nil {
		return fmt.Errorf("failed to get a token to register: %w", err)
	}
	if err := client.Register(ctx, r, value, userID); err != nil {
		return fmt.Errorf("failed to register %s with the FUSE daemons: %w", r, err)
	}
	return nil
}

func refreshUntilDone(ctx context.Context, client *Client, r Registration, token TokenFunc, userID string) {
	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			revoke(ctx, client, r)
			return
		case <-ticker.C:
			if err := register(ctx, client, r, token, userID); err != nil {
				// The next tick retries. Until one succeeds the daemons keep the
				// previous registration, so access survives a transient failure
				// and only a token expiry is at risk.
				log.Warnf(ctx, "Failed to refresh the FUSE registration, retrying in %v: %v", RefreshInterval, err)
				continue
			}
			log.Infof(ctx, "Refreshed the FUSE registration for %s", r)
		}
	}
}

// revoke clears the registration on the way out. A stale one is harmless — the
// daemons compare the start time, so a recycled pid cannot inherit it — but
// revoking keeps their tables bounded on a long-lived driver.
func revoke(ctx context.Context, client *Client, r Registration) {
	// ctx is what triggered this, so the revoke needs a context of its own.
	revokeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*requestTimeout)
	defer cancel()

	if err := client.Revoke(revokeCtx, r); err != nil {
		log.Warnf(ctx, "Failed to revoke the FUSE registration for %s: %v", r, err)
		return
	}
	log.Infof(ctx, "Revoked the FUSE registration for %s", r)
}
