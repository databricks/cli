package filer

import (
	"context"
	"net/http"

	tmpauth "github.com/databricks/cli/libs/tmp/auth"
	"github.com/databricks/databricks-sdk-go/config"
)

// configCredentials adapts the CLI's resolved SDK config into the credentials
// interface expected by the vendored files/v2 client. It signs a throwaway
// request with config.Authenticate and hands the resulting headers to the
// client, so the files/v2 client reuses the exact auth the rest of the CLI
// already resolved instead of re-reading a profile.
type configCredentials struct {
	cfg *config.Config
}

func (c configCredentials) Name() string {
	return "databricks-cli"
}

func (c configCredentials) AuthHeaders(ctx context.Context) ([]tmpauth.Header, error) {
	// The URL is irrelevant: config.Authenticate only reads it to pick the auth
	// scheme, and every files/v2 request targets the same workspace host.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.Host, nil)
	if err != nil {
		return nil, err
	}
	if err := c.cfg.Authenticate(req); err != nil {
		return nil, err
	}

	headers := make([]tmpauth.Header, 0, len(req.Header))
	for key, values := range req.Header {
		for _, value := range values {
			headers = append(headers, tmpauth.Header{Key: key, Value: value})
		}
	}
	return headers, nil
}
