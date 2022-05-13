package project

import (
	"context"

	"github.com/databrickslabs/terraform-provider-databricks/common"
	"github.com/databrickslabs/terraform-provider-databricks/commands"
)


type appContext int

const (
	// DatabricksClient holds DatabricksClient
	DatabricksClient appContext = 1
)

func Authenticate(ctx context.Context) context.Context {
	client := common.CommonEnvironmentClient()
	client.WithCommandExecutor(func(ctx context.Context, _ *common.DatabricksClient) common.CommandExecutor {
		return commands.NewCommandsAPI(ctx, client)
	})
	return context.WithValue(ctx, DatabricksClient, client)
}

func ClientFromContext(ctx context.Context) *common.DatabricksClient {
	client, ok := ctx.Value(DatabricksClient).(*common.DatabricksClient)
	if !ok {
		panic("authentication is not configured")
	}
	return client
}
