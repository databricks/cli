package filer

import (
	"context"
	"net/http"

	"github.com/databricks/databricks-sdk-go/config"
	sdkauth "github.com/databricks/sdk-go/auth"
)

// configCredentials adapts the CLI's resolved SDK config into the credentials
// interface expected by the files/v2 client. It signs a throwaway
// request with config.Authenticate and hands the resulting headers to the
// client, so the files/v2 client reuses the exact auth the rest of the CLI
// already resolved instead of re-reading a profile.
type configCredentials struct {
	cfg *config.Config
}

// Name reports the resolved auth mechanism (e.g. "pat", "oauth-m2m"), which the
// files/v2 client folds into its User-Agent. The CLI resolves auth when it
// builds the workspace client, so AuthType is normally set before the filer is
// created; fall back to "unk" when it is empty because the files/v2 client
// rejects an empty auth name with an "invalid value" error at construction.
func (c configCredentials) Name() string {
	if c.cfg.AuthType == "" {
		return "unk"
	}
	return c.cfg.AuthType
}

func (c configCredentials) AuthHeaders(ctx context.Context) ([]sdkauth.Header, error) {
	// The URL is irrelevant: config.Authenticate only reads it to pick the auth
	// scheme, and every files/v2 request targets the same workspace host.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.Host, nil)
	if err != nil {
		return nil, err
	}
	if err := c.cfg.Authenticate(req); err != nil {
		return nil, err
	}

	headers := make([]sdkauth.Header, 0, len(req.Header))
	for key, values := range req.Header {
		for _, value := range values {
			headers = append(headers, sdkauth.Header{Key: key, Value: value})
		}
	}
	return headers, nil
}
