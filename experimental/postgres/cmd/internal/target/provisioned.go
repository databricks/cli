package target

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	database "github.com/databricks/sdk-go/database/v1"
	"github.com/google/uuid"
)

// GetProvisioned fetches a single provisioned instance by name.
// The Name field on the response can be empty; this function ensures it is
// populated from the input so downstream callers do not have to re-set it.
func GetProvisioned(ctx context.Context, w *sdkclients.Clients, name string) (*database.DatabaseInstance, error) {
	instance, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{Name: new(name)})
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	if utils.Deref(instance.Name) == "" {
		instance.Name = new(name)
	}
	return instance, nil
}

// ProvisionedCredential issues a short-lived OAuth token for the provisioned
// instance with the given name.
func ProvisionedCredential(ctx context.Context, w *sdkclients.Clients, instanceName string) (string, error) {
	cred, err := w.Database.GenerateDatabaseCredential(ctx, database.GenerateDatabaseCredentialRequest{
		InstanceNames: []string{instanceName},
		RequestId:     new(uuid.NewString()),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get database credentials: %w", err)
	}
	return utils.Deref(cred.Token), nil
}
