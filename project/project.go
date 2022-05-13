package project

import (
	"context"
	"fmt"

	"github.com/databrickslabs/terraform-provider-databricks/clusters"
	"github.com/databrickslabs/terraform-provider-databricks/common"
	"github.com/databrickslabs/terraform-provider-databricks/scim"
)

func CurrentUser(ctx context.Context) (scim.User, error) {
	// TODO: memoize
	return scim.NewUsersAPI(ctx, ClientFromContext(ctx)).Me()
}

func ProjectName(ctx context.Context) string {
	return "dev" // TODO: parse from config file
}

func DevelopmentCluster(ctx context.Context) (cluster clusters.ClusterInfo, err error) {
	api := clusters.NewClustersAPI(ctx, ClientFromContext(ctx)) // TODO: rewrite with normal SDK
	me, err := CurrentUser(ctx)
	if err != nil {
		return
	}
	projectName := ProjectName(ctx)
	devClusterName := fmt.Sprintf("dev/%s/%s", projectName, me.UserName)
	return api.GetOrCreateRunningCluster(devClusterName)
}

func runCommandOnDev(ctx context.Context, language, command string) common.CommandResults {
	client := ClientFromContext(ctx)
	exec := client.CommandExecutor(ctx)
	cluster, err := DevelopmentCluster(ctx)
	if err != nil {
		return common.CommandResults{
			ResultType: "error",
			Summary: err.Error(),
		}
	}
	return exec.Execute(cluster.ClusterID, language, command)
}

func RunPythonOnDev(ctx context.Context, command string) common.CommandResults {
	return runCommandOnDev(ctx, "python", command)
}