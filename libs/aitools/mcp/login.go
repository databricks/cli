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

// SignInPollInterval and SignInWaitTimeout are the polling settings of provider sign-in (SPEC 12).
const (
	SignInPollInterval = 2 * time.Second
	SignInWaitTimeout  = 5 * time.Minute
)

// ErrSignInTimeout means the caller's credential did not become ACTIVE before the wait ended.
var ErrSignInTimeout = errors.New("sign-in did not complete before the timeout")

// ErrSignInFailed means the credential reached the FAILED provisioning state.
var ErrSignInFailed = errors.New("sign-in failed")

// SignInOptions controls how SignIn waits. Callers set both durations.
type SignInOptions struct {
	Poll    time.Duration
	Timeout time.Duration
	// Open opens the sign-in page. Nil leaves the page to the user. It is not called when
	// BROWSER=none, because the URL is already printed.
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

// SignIn signs the caller in to a sign-in-required service. It returns nil at once when the
// caller is already signed in. Otherwise it prints the sign-in URL to out, opens it, and polls
// until the credential is ACTIVE. The errors are ErrSignInTimeout, ErrSignInFailed, or an API error.
func SignIn(ctx context.Context, w *databricks.WorkspaceClient, host, workspaceID, service string, out io.Writer, opts SignInOptions) error {
	state, err := CredentialState(ctx, w, service)
	if err != nil {
		return err
	}
	if state == catalog.ProvisioningInfoStateActive {
		return nil
	}

	url := SignInURL(host, service, workspaceID)
	fmt.Fprintf(out, "Open this URL to sign in to %s:\n%s\n", service, url)
	if opts.Open != nil && !browser.IsDisabled(ctx) {
		if err := opts.Open(ctx, url); err != nil {
			fmt.Fprintf(out, "Could not open a browser (%v). Open the URL above.\n", err)
		}
	}
	fmt.Fprintln(out, "Waiting for sign-in...")

	deadline := time.NewTimer(opts.Timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(opts.Poll)
	defer ticker.Stop()

	// The deadline is a timer, not a context deadline: the SDK rate limiter refuses a request
	// whose context would expire before it may run, which would hide the timeout.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return ErrSignInTimeout
		case <-ticker.C:
		}

		state, err := CredentialState(ctx, w, service)
		if err != nil {
			return err
		}
		// Other states (PROVISIONING, UPDATING, ...) mean the login has not completed yet.
		if state == catalog.ProvisioningInfoStateActive {
			return nil
		}
		if state == catalog.ProvisioningInfoStateFailed {
			return ErrSignInFailed
		}
	}
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
