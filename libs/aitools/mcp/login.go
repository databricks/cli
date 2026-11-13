package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// SignInPollInterval and SignInWaitTimeout are the polling settings of provider sign-in.
const (
	SignInPollInterval = 2 * time.Second
	SignInWaitTimeout  = 5 * time.Minute
)

// ErrSignInTimeout means the caller's credential did not become ACTIVE before the wait ended.
var ErrSignInTimeout = errors.New("sign-in did not complete before the timeout")

// ErrSignInFailed means the credential reached the FAILED provisioning state.
var ErrSignInFailed = errors.New("sign-in failed")

// SignInOptions controls how SignIn waits. Callers set all three fields.
type SignInOptions struct {
	Poll    time.Duration
	Timeout time.Duration
	// Open opens the sign-in page. It is not called when BROWSER=none, because the URL is already printed.
	Open func(ctx context.Context, url string) error
}

// CredentialState returns the provisioning state of the caller's sign-in to a sign-in-required
// service. It is "" when the caller has not signed in: the API answers NOT_FOUND until then.
func CredentialState(ctx context.Context, w *databricks.WorkspaceClient, service string) (catalog.ProvisioningInfoState, error) {
	cred, err := w.AiGateway.GetMcpServiceUserMappedCredential(ctx, catalog.GetMcpServiceUserMappedCredentialRequest{
		Name: mcpServicePrefix + service,
	})
	if errors.Is(err, apierr.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if cred.ProvisioningInfo == nil {
		return "", nil
	}
	return cred.ProvisioningInfo.State, nil
}

// SignIn signs the caller in to a sign-in-required service, at url from SignInURL. It returns nil at once when the
// caller is already signed in. Otherwise it prints url to out, opens it, and polls until the credential is ACTIVE.
// The errors are ErrSignInTimeout, ErrSignInFailed, or an API error.
func SignIn(ctx context.Context, w *databricks.WorkspaceClient, url, service string, out io.Writer, opts SignInOptions) error {
	state, err := CredentialState(ctx, w, service)
	if err != nil {
		return err
	}
	if state == catalog.ProvisioningInfoStateActive {
		return nil
	}

	fmt.Fprintf(out, "Open this URL to sign in to %s:\n%s\n", service, url)
	if !browser.IsDisabled(ctx) {
		if err := opts.Open(ctx, url); err != nil {
			fmt.Fprintf(out, "Could not open a browser (%v). Open the URL above.\n", err)
		}
	}
	fmt.Fprintln(out, "Waiting for sign-in...")

	// The timeout cancels the wait from a timer, not with a context deadline: the SDK rate limiter refuses a request
	// whose wait would exceed a deadline, which would hide the timeout. A poll still in flight is cut off too, so the
	// SDK's retries of a slow poll cannot hold the command past the timeout.
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	timer := time.AfterFunc(opts.Timeout, cancel)
	defer timer.Stop()
	ticker := time.NewTicker(opts.Poll)
	defer ticker.Stop()

	// A FAILED state left by an earlier attempt is still there when this one starts. FAILED counts only once the
	// state has changed since the previous read, so it is the result of this attempt.
	previous := state
	for {
		select {
		case <-waitCtx.Done():
			return waitEnded(ctx)
		case <-ticker.C:
		}

		state, err := CredentialState(waitCtx, w, service)
		if err != nil {
			if waitCtx.Err() != nil {
				return waitEnded(ctx)
			}
			return err
		}
		// Other states (PROVISIONING, UPDATING, ...) mean the login has not completed yet.
		if state == catalog.ProvisioningInfoStateActive {
			return nil
		}
		if state == catalog.ProvisioningInfoStateFailed && state != previous {
			return ErrSignInFailed
		}
		previous = state
	}
}

// waitEnded returns why the sign-in wait ended: the caller's cancellation, or else the timeout.
func waitEnded(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrSignInTimeout
}

// SignOut deletes the caller's sign-in to a service. It reports false when the caller was not
// signed in, which is not an error.
func SignOut(ctx context.Context, w *databricks.WorkspaceClient, service string) (bool, error) {
	_, err := w.AiGateway.DeleteMcpServiceUserMappedCredential(ctx, catalog.DeleteMcpServiceUserMappedCredentialRequest{
		Name: mcpServicePrefix + service,
	})
	if errors.Is(err, apierr.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
