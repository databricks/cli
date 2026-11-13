package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// SignInTimeoutError means the caller's credential did not become ACTIVE before the wait ended. LastCheck is the
// failure of the last poll, when that poll failed; a poll that reads the credential clears it.
type SignInTimeoutError struct {
	LastCheck error
}

func (e *SignInTimeoutError) Error() string {
	return "sign-in did not complete before the timeout"
}

func (e *SignInTimeoutError) Unwrap() error {
	return e.LastCheck
}

// ErrSignInFailed means the credential reached the FAILED provisioning state.
var ErrSignInFailed = errors.New("sign-in failed")

// SignInOptions controls how SignIn waits. Poll and Timeout are required. Profile names the profile in the notice
// before the page. Open opens the page in the background, and is not called when BROWSER=none, because the URL is
// already printed.
type SignInOptions struct {
	Profile string
	Poll    time.Duration
	Timeout time.Duration
	Open    func(ctx context.Context, url string) error
}

// CredentialState returns the provisioning state of the caller's sign-in to a sign-in-required
// service. It is "" when the caller has not signed in: the API answers NOT_FOUND until then.
func CredentialState(ctx context.Context, w *databricks.WorkspaceClient, service string) (catalog.ProvisioningInfoState, error) {
	cred, err := w.AiGateway.GetMcpServiceUserMappedCredential(ctx, catalog.GetMcpServiceUserMappedCredentialRequest{
		Name: serviceResourceName(service),
	})
	if IsNotFound(err) {
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

// SignIn signs the caller in to a sign-in-required service, at url from SignInURL. When the caller is already signed in
// it returns alreadySignedIn true and nil, and prints nothing and opens no page. Otherwise it deletes a FAILED credential
// left by an earlier attempt, prints url to out, opens it, and polls until the credential is ACTIVE. A poll that fails
// with a server error or a network error is polled again until the wait ends; any other failed poll ends the sign-in.
// The errors are a *SignInTimeoutError, ErrSignInFailed, or an API error.
func SignIn(ctx context.Context, w *databricks.WorkspaceClient, url, service string, out io.Writer, opts SignInOptions) (alreadySignedIn bool, err error) {
	state, err := CredentialState(ctx, w, service)
	if err != nil {
		return false, err
	}
	switch state {
	case catalog.ProvisioningInfoStateActive:
		return true, nil
	case catalog.ProvisioningInfoStateFailed:
		// A FAILED credential is left by an earlier attempt. It is deleted, so that a FAILED state seen while waiting is
		// the result of this attempt, and so that the page starts from no credential.
		if _, err := SignOut(ctx, w, service); err != nil {
			return false, err
		}
	case catalog.ProvisioningInfoStateDegraded, catalog.ProvisioningInfoStateDeleting,
		catalog.ProvisioningInfoStateProvisioning, catalog.ProvisioningInfoStateUpdating:
		// Not a state this sign-in starts from; the wait below handles it.
	}

	fmt.Fprintf(out, "Sign in with the same account you use for Databricks profile %s.\n", opts.Profile)
	fmt.Fprintf(out, "Open this URL to sign in to %s:\n%s\n", service, url)
	// The page opens in the background. A browser command can run for as long as the browser is open, and the wait does
	// not depend on it. Its failure is reported when it returns, if that is before the sign-in completes.
	var opened <-chan error
	if !browser.IsDisabled(ctx) {
		done := make(chan error, 1)
		go func() { done <- opts.Open(ctx, url) }()
		opened = done
	}
	fmt.Fprintln(out, "Waiting for sign-in...")

	// The timeout cancels the wait from a timer, not with a context deadline: the SDK's rate limiter refuses a request
	// whose wait would exceed a deadline, which would hide the timeout (see
	// https://github.com/databricks/databricks-sdk-go/blob/v0.197.0/httpclient/api_client.go and
	// https://github.com/golang/time/blob/v0.14.0/rate/rate.go). A poll still in flight is cut off too, so the SDK's
	// retries of a slow poll cannot hold the command past the timeout.
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	timer := time.AfterFunc(opts.Timeout, cancel)
	defer timer.Stop()
	ticker := time.NewTicker(opts.Poll)
	defer ticker.Stop()

	// lastCheck is the failure of the last poll, if that poll failed. The timeout reports it.
	var lastCheck error
	for {
		select {
		case <-waitCtx.Done():
			return false, waitEnded(ctx, lastCheck)
		case err := <-opened:
			opened = nil
			if err != nil {
				fmt.Fprintf(out, "Could not open a browser (%v). Open the URL above.\n", err)
			}
		case <-ticker.C:
		}

		state, err := CredentialState(waitCtx, w, service)
		if err != nil {
			if waitCtx.Err() != nil {
				return false, waitEnded(ctx, lastCheck)
			}
			if !transientPollError(err) {
				return false, err
			}
			lastCheck = err
			continue
		}
		lastCheck = nil
		// Other states (PROVISIONING, UPDATING, ...) mean the login has not completed yet.
		switch state {
		case catalog.ProvisioningInfoStateActive:
			return false, nil
		case catalog.ProvisioningInfoStateFailed:
			return false, ErrSignInFailed
		case catalog.ProvisioningInfoStateDegraded, catalog.ProvisioningInfoStateDeleting,
			catalog.ProvisioningInfoStateProvisioning, catalog.ProvisioningInfoStateUpdating:
			// The login has not completed yet.
		}
	}
}

// transientPollError reports whether a failed poll is worth polling again: a server error, or a network failure. Any
// other error, such as a refused credential, ends the sign-in.
func transientPollError(err error) bool {
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		return apiErr.StatusCode >= http.StatusInternalServerError
	}
	_, ok := errors.AsType[*url.Error](err)
	return ok
}

// waitEnded returns why the sign-in wait ended: the caller's cancellation, or else the timeout, with the failure of the
// last poll.
func waitEnded(ctx context.Context, lastCheck error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return &SignInTimeoutError{LastCheck: lastCheck}
}

// SignOut deletes the caller's sign-in to a service. It reports false when the caller was not
// signed in, which is not an error.
func SignOut(ctx context.Context, w *databricks.WorkspaceClient, service string) (bool, error) {
	_, err := w.AiGateway.DeleteMcpServiceUserMappedCredential(ctx, catalog.DeleteMcpServiceUserMappedCredentialRequest{
		Name: serviceResourceName(service),
	})
	if IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
