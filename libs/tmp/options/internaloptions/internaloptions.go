// Package internaloptions contains the resolved options types produced
// by applying [github.com/databricks/cli/libs/tmp/options/client.Option] and
// [github.com/databricks/cli/libs/tmp/options/call.Option] values.
//
// IMPORTANT: This package is NOT part of the public API of the Databricks
// SDK. Its contents may change at any time without notice. Clients should
// not directly depend on functionalities in this package.
package internaloptions

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/tmp/auth"
	"github.com/databricks/sdk-go/core/ops"
	"github.com/databricks/sdk-go/core/profiles"
)

// ClientOptions is the resolved client configuration produced by applying
// client.Option values.
type ClientOptions struct {
	// Profile resolution.
	ProfileName              string
	ProfileFile              string
	DisableProfileResolution bool

	Host        string
	AccountID   string
	WorkspaceID string

	Credentials auth.Credentials

	Timeout    time.Duration
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// Resolve fills in defaults and validates the resolved client options.
//
// Resolve always populates the HTTPClient and Logger fields with default
// values if not provided.
func (c *ClientOptions) Resolve() error {
	if c.Logger == nil {
		c.Logger = slog.New(slog.DiscardHandler)
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: c.Timeout,
			// Share the default transport so the connection pool is reused
			// across all clients.
			Transport: http.DefaultTransport,
		}
	}

	if err := c.resolve(); err != nil {
		return err
	}
	return c.validate()
}

// resolve fills unset options from the profile. Explicitly set options take
// precedence and are never overwritten.
//
// TODO: Apply environment-variable overrides, and resolve workspace/account ID
// and credentials from the profile when not provided.
func (c *ClientOptions) resolve() error {
	if c.DisableProfileResolution {
		return nil
	}

	var opts []profiles.ResolveOption
	if c.ProfileName != "" {
		opts = append(opts, profiles.WithProfile(c.ProfileName))
	} else {
		opts = append(opts, profiles.WithDefaultProfile())
	}
	if c.ProfileFile != "" {
		opts = append(opts, profiles.WithFile(c.ProfileFile))
	}

	p, err := profiles.Resolve(opts...)
	if err != nil {
		return err
	}

	if c.Host == "" {
		c.Host = p.Host
	}
	if c.AccountID == "" {
		c.AccountID = p.AccountID
	}
	if c.WorkspaceID == "" {
		c.WorkspaceID = p.WorkspaceID
	}
	return nil
}

func (c *ClientOptions) validate() error {
	if c.Credentials == nil {
		return errors.New("credentials are required")
	}
	return nil
}

// CallOptions is the resolved per-call configuration produced by applying
// call.Option values.
type CallOptions struct {
	Retrier     func() ops.Retrier
	RateLimiter ops.Limiter
	Timeout     time.Duration
}
