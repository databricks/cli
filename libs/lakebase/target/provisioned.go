package target

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"
)

// ListProvisionedInstances returns all provisioned database instances in the workspace.
func ListProvisionedInstances(ctx context.Context, w *databricks.WorkspaceClient) ([]database.DatabaseInstance, error) {
	return w.Database.ListDatabaseInstancesAll(ctx, database.ListDatabaseInstancesRequest{})
}

// GetProvisioned fetches a single provisioned instance by name.
// The Name field on the response can be empty; this function ensures it is
// populated from the input so downstream callers do not have to re-set it.
func GetProvisioned(ctx context.Context, w *databricks.WorkspaceClient, name string) (*database.DatabaseInstance, error) {
	instance, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	if instance.Name == "" {
		instance.Name = name
	}
	return instance, nil
}

// AutoSelectProvisioned returns the only provisioned instance in the workspace,
// or an AmbiguousError if there are multiple. Returns a plain error if none.
func AutoSelectProvisioned(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	instances, err := ListProvisionedInstances(ctx, w)
	if err != nil {
		return "", err
	}
	if len(instances) == 0 {
		return "", errors.New("no Lakebase Provisioned instances found in workspace")
	}
	if len(instances) == 1 {
		return instances[0].Name, nil
	}

	choices := make([]Choice, 0, len(instances))
	for _, inst := range instances {
		choices = append(choices, Choice{ID: inst.Name, DisplayName: inst.Name})
	}
	return "", &AmbiguousError{Kind: "instance", FlagHint: "--target", Choices: choices}
}

// ProvisionedCredential issues a short-lived OAuth token for the provisioned
// instance with the given name.
func ProvisionedCredential(ctx context.Context, w *databricks.WorkspaceClient, instanceName string) (string, error) {
	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{instanceName},
		RequestId:     uuid.NewString(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get database credentials: %w", err)
	}
	return cred.Token, nil
}
