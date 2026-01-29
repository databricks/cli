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

// GetEndpoint retrieves an endpoint by its full resource name.
func GetEndpoint(ctx context.Context, w *databricks.WorkspaceClient, projectID, branchID, endpointID string) (*postgres.Endpoint, error) {
	endpointName := fmt.Sprintf("projects/%s/branches/%s/endpoints/%s", projectID, branchID, endpointID)
	endpoint, err := w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{
		Name: endpointName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint %s: %w", endpointName, err)
	}
	return endpoint, nil
}

// Connect connects to a Postgres endpoint with retry logic.
func Connect(ctx context.Context, w *databricks.WorkspaceClient, endpoint *postgres.Endpoint, retryConfig psql.RetryConfig, extraArgs ...string) error {
	endpointID := ExtractIDFromName(endpoint.Name, "endpoints")
	cmdio.LogString(ctx, fmt.Sprintf("Connecting to Postgres endpoint %s ...", endpointID))

	user, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("error getting current user: %w", err)
	}

	if endpoint.Status == nil {
		return errors.New("endpoint status is not available")
	}

	state := endpoint.Status.CurrentState
	cmdio.LogString(ctx, fmt.Sprintf("Endpoint state: %s", state))

	if state != postgres.EndpointStatusStateActive && state != postgres.EndpointStatusStateIdle {
		if state == postgres.EndpointStatusStateInit {
			cmdio.LogString(ctx, "Please retry when the endpoint becomes active")
		}
		return errors.New("endpoint is not ready for accepting connections")
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
	cmdio.LogString(ctx, "Successfully fetched database credentials")

	return psql.Connect(ctx, psql.ConnectOptions{
		Host:            host,
		Username:        user.UserName,
		Password:        cred.Token,
		DefaultDatabase: "postgres",
		ExtraArgs:       extraArgs,
	}, retryConfig)
}
