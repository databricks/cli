package dresources

import (
	"context"

	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type ResourceStorageCredential struct {
	client *databricks.WorkspaceClient
}

func (*ResourceStorageCredential) New(client *databricks.WorkspaceClient) *ResourceStorageCredential {
	return &ResourceStorageCredential{client: client}
}

func (*ResourceStorageCredential) PrepareState(input *resources.StorageCredential) *catalog.CreateStorageCredential {
	return &input.CreateStorageCredential
}

func (*ResourceStorageCredential) RemapState(info *catalog.StorageCredentialInfo) *catalog.CreateStorageCredential {
	out := &catalog.CreateStorageCredential{
		Comment:         info.Comment,
		Name:            info.Name,
		ReadOnly:        info.ReadOnly,
		ForceSendFields: utils.FilterFields[catalog.CreateStorageCredential](info.ForceSendFields),
	}
	if info.AwsIamRole != nil {
		out.AwsIamRole = &catalog.AwsIamRoleRequest{RoleArn: info.AwsIamRole.RoleArn}
	}
	if info.AzureManagedIdentity != nil {
		out.AzureManagedIdentity = &catalog.AzureManagedIdentityRequest{
			AccessConnectorId: info.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: info.AzureManagedIdentity.ManagedIdentityId,
		}
	}
	if info.AzureServicePrincipal != nil {
		// UC does not echo the client_secret on read; preserve it from input.
		out.AzureServicePrincipal = &catalog.AzureServicePrincipal{
			DirectoryId:   info.AzureServicePrincipal.DirectoryId,
			ApplicationId: info.AzureServicePrincipal.ApplicationId,
		}
	}
	if info.DatabricksGcpServiceAccount != nil {
		out.DatabricksGcpServiceAccount = &catalog.DatabricksGcpServiceAccountRequest{}
	}
	if info.CloudflareApiToken != nil {
		out.CloudflareApiToken = &catalog.CloudflareApiToken{
			AccountId:    info.CloudflareApiToken.AccountId,
			AccessKeyId:  info.CloudflareApiToken.AccessKeyId,
			SecretAccessKey: info.CloudflareApiToken.SecretAccessKey,
		}
	}
	return out
}

func (r *ResourceStorageCredential) DoRead(ctx context.Context, id string) (*catalog.StorageCredentialInfo, error) {
	return r.client.StorageCredentials.GetByName(ctx, id)
}

func (r *ResourceStorageCredential) DoCreate(ctx context.Context, config *catalog.CreateStorageCredential) (string, *catalog.StorageCredentialInfo, error) {
	response, err := r.client.StorageCredentials.Create(ctx, *config)
	if err != nil || response == nil {
		return "", nil, err
	}
	return response.Name, response, nil
}

func (r *ResourceStorageCredential) DoUpdate(ctx context.Context, id string, config *catalog.CreateStorageCredential, _ *PlanEntry) (*catalog.StorageCredentialInfo, error) {
	updateRequest := catalog.UpdateStorageCredential{
		Comment:                     config.Comment,
		Name:                        id,
		ReadOnly:                    config.ReadOnly,
		Owner:                       "",
		AwsIamRole:                  config.AwsIamRole,
		AzureServicePrincipal:       config.AzureServicePrincipal,
		CloudflareApiToken:          config.CloudflareApiToken,
		DatabricksGcpServiceAccount: config.DatabricksGcpServiceAccount,
		ForceSendFields:             utils.FilterFields[catalog.UpdateStorageCredential](config.ForceSendFields, "Owner"),
	}

	if config.AzureManagedIdentity != nil {
		// Update request uses AzureManagedIdentityResponse for read-only echo
		// of access_connector_id; map the create request type into it.
		updateRequest.AzureManagedIdentity = &catalog.AzureManagedIdentityResponse{
			AccessConnectorId: config.AzureManagedIdentity.AccessConnectorId,
			ManagedIdentityId: config.AzureManagedIdentity.ManagedIdentityId,
		}
	}

	return r.client.StorageCredentials.Update(ctx, updateRequest)
}

func (r *ResourceStorageCredential) DoDelete(ctx context.Context, id string) error {
	return r.client.StorageCredentials.DeleteByName(ctx, id)
}
