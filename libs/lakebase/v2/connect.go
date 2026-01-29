package lakebasev2

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/lakebase/psql"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// ExtractIDFromName extracts the ID component from a resource name.
// For example, ExtractIDFromName("projects/foo/branches/bar", "branches") returns "bar".
func ExtractIDFromName(name, component string) string {
	parts := strings.Split(name, "/")
	for i := range len(parts) - 1 {
		if parts[i] == component {
			return parts[i+1]
		}
	}
	return name
}

// FormatEndpointType returns a human-readable endpoint type string.
func FormatEndpointType(endpointType postgres.EndpointType) string {
	switch endpointType {
	case postgres.EndpointTypeEndpointTypeReadWrite:
		return "read-write"
	case postgres.EndpointTypeEndpointTypeReadOnly:
		return "read-only"
	default:
		return string(endpointType)
	}
}

// FormatEndpointState returns a human-readable state description.
func FormatEndpointState(state postgres.EndpointStatusState) string {
	switch state {
	case postgres.EndpointStatusStateActive:
		return ""
	case postgres.EndpointStatusStateIdle:
		return "idle, waking up"
	case postgres.EndpointStatusStateInit:
		return "initializing"
	default:
		return strings.ToLower(string(state))
	}
}

// Connect connects to a Postgres endpoint with retry logic.
func Connect(ctx context.Context, w *databricks.WorkspaceClient, endpoint *postgres.Endpoint, retryConfig psql.RetryConfig, extraArgs ...string) error {
	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("error getting current user: %w", err)
	}

	if endpoint.Status == nil {
		return errors.New("endpoint status is not available")
	}

	state := endpoint.Status.CurrentState
	if state != postgres.EndpointStatusStateActive && state != postgres.EndpointStatusStateIdle {
		if state == postgres.EndpointStatusStateInit {
			cmdio.LogString(ctx, "Endpoint is initializing, please retry when it becomes active")
		}
		return errors.New("endpoint is not ready for accepting connections")
	}

	// Log connection details
	endpointType := FormatEndpointType(endpoint.Status.EndpointType)
	stateDesc := FormatEndpointState(state)
	if stateDesc != "" {
		cmdio.LogString(ctx, fmt.Sprintf("Connecting to %s endpoint (%s)...", endpointType, stateDesc))
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("Connecting to %s endpoint...", endpointType))
	}

	if endpoint.Status.Hosts == nil || endpoint.Status.Hosts.Host == "" {
		return errors.New("endpoint host information is not available")
	}
	host := endpoint.Status.Hosts.Host

	cred, err := w.Postgres.GenerateDatabaseCredential(ctx, postgres.GenerateDatabaseCredentialRequest{
		Endpoint: endpoint.Name,
	})
	if err != nil {
		return fmt.Errorf("error getting database credentials: %w", err)
	}

	return psql.Connect(ctx, psql.ConnectOptions{
		Host:            host,
		Username:        user.UserName,
		Password:        cred.Token,
		DefaultDatabase: "postgres",
		ExtraArgs:       extraArgs,
	}, retryConfig)
}
