package sync

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
)

// DefaultRetryTimeout bounds how long the sync layer keeps retrying transient
// gateway errors per filer call. The SDK only retries 429 and 504
// (httpclient/errors.go DefaultErrorRetriable); 502 and 503 land here.
const DefaultRetryTimeout = 30 * time.Second

func isTransientGatewayError(err error) bool {
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return false
	}
	switch aerr.StatusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryOnTransient runs fn, retrying transient gateway errors
// (HTTP 502/503/504) until timeout elapses. Backoff and jitter are provided
// by retries.Poll.
func retryOnTransient(ctx context.Context, timeout time.Duration, label string, fn func() error) error {
	if timeout <= 0 {
		return fn()
	}
	_, err := retries.Poll(ctx, timeout, func() (*struct{}, *retries.Err) {
		err := fn()
		if err == nil {
			return nil, nil
		}
		if !isTransientGatewayError(err) {
			return nil, retries.Halt(err)
		}
		log.Warnf(ctx, "sync %s: retrying after transient error: %s", label, err)
		return nil, retries.Continue(err)
	})
	return err
}
