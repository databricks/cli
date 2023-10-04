package apps

import "context"

type AppsService interface {
	Deploy(ctx context.Context, request *DeployAppRequest) (*DeploymentResponse, error)

	Delete(ctx context.Context, request *DeleteAppRequest) (*DeleteResponse, error)

	Get(ctx context.Context, request *GetAppRequest) (*GetResponse, error)
}
