package dresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type ResourcePermissions struct {
	client *databricks.WorkspaceClient
}

type PermissionsState struct {
	ObjectID    string                     `json:"object_id"`
	Permissions []iam.AccessControlRequest `json:"permissions,omitempty"`
}

func PreparePermissionsInputConfig(inputConfig any, node string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(node, ".permissions")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .permissions", node)
	}
	switch v := inputConfig.(type) {
	case *[]resources.JobPermission:
		return initStructVar("/jobs/", baseNode, *v), nil
	case *[]resources.PipelinePermission:
		return initStructVar("/pipelines/", baseNode, *v), nil
	case *[]resources.AppPermission:
		return initStructVar("/apps/", baseNode, *v), nil
	case *[]resources.ClusterPermission:
		return initStructVar("/clusters/", baseNode, *v), nil
	case *[]resources.DatabaseInstancePermission:
		return initStructVar("/database-instances/", baseNode, *v), nil
	case *[]resources.DashboardPermission:
		return initStructVar("/dashboards/", baseNode, *v), nil
	case *[]resources.MlflowExperimentPermission:
		return initStructVar("/experiments/", baseNode, *v), nil
	case *[]resources.MlflowModelPermission:
		return initStructVar("/registered-models/", baseNode, *v), nil
	case *[]resources.ModelServingEndpointPermission:
		return initStructVar("/serving-endpoints/", baseNode, *v), nil
	case *[]resources.SqlWarehousePermission:
		return initStructVar("/sql/warehouses/", baseNode, *v), nil
	default:
		return nil, fmt.Errorf("unsupported type for permissions: %T", inputConfig)
	}
}

func initStructVar[T resources.IPermission](prefix, baseNode string, v []T) *structvar.StructVar {
	permissions := make([]iam.AccessControlRequest, 0, len(v))

	for _, p := range v {
		permissions = append(permissions, iam.AccessControlRequest{
			PermissionLevel:      iam.PermissionLevel(p.GetLevel()),
			GroupName:            p.GetGroupName(),
			ServicePrincipalName: p.GetServicePrincipalName(),
			UserName:             p.GetUserName(),
			ForceSendFields:      nil,
		})
	}

	return &structvar.StructVar{
		Config: &PermissionsState{
			ObjectID:    "", // Always a reference, defined in Refs below
			Permissions: permissions,
		},
		Refs: map[string]string{
			"object_id": prefix + "${" + baseNode + ".id}",
		},
	}
}

func (*ResourcePermissions) New(client *databricks.WorkspaceClient) *ResourcePermissions {
	return &ResourcePermissions{client: client}
}

func (*ResourcePermissions) PrepareState(s *PermissionsState) *PermissionsState {
	return s
}

// parsePermissionsID extracts the object type and ID from a permissions ID string.
// Handles both 3-part IDs ("/jobs/123") and 4-part IDs ("/sql/warehouses/uuid").
func parsePermissionsID(id string) (extractedType, extractedID string, err error) {
	idParts := strings.Split(id, "/")
	if len(idParts) < 3 { // need at least "/type/id"
		return "", "", fmt.Errorf("cannot parse id: %q", id)
	}

	if len(idParts) == 3 { // "/jobs/123"
		extractedType = idParts[1]
		extractedID = idParts[2]
	} else if len(idParts) == 4 { // "/sql/warehouses/uuid"
		extractedType = idParts[1] + "/" + idParts[2] // "sql/warehouses"
		extractedID = idParts[3]
	} else {
		return "", "", fmt.Errorf("cannot parse id: %q", id)
	}

	return extractedType, extractedID, nil
}

func (r *ResourcePermissions) DoRefresh(ctx context.Context, id string) (*PermissionsState, error) {
	extractedType, extractedID, err := parsePermissionsID(id)
	if err != nil {
		return nil, err
	}

	acl, err := r.client.Permissions.Get(ctx, iam.GetPermissionRequest{
		RequestObjectId:   extractedID,
		RequestObjectType: extractedType,
	})
	if err != nil {
		return nil, err
	}

	result := PermissionsState{
		ObjectID:    id,
		Permissions: nil,
	}

	for _, accessControl := range acl.AccessControlList {
		for _, permission := range accessControl.AllPermissions {
			// Inherited permissions can be ignored, as they are not set by the user (following TF)
			if permission.Inherited {
				continue
			}
			result.Permissions = append(result.Permissions, iam.AccessControlRequest{
				GroupName:            accessControl.GroupName,
				UserName:             accessControl.UserName,
				ServicePrincipalName: accessControl.ServicePrincipalName,
				PermissionLevel:      permission.PermissionLevel,
				ForceSendFields:      nil,
			})
		}
	}

	return &result, nil
}

// DoCreate calls https://docs.databricks.com/api/workspace/jobs/setjobpermissions.
func (r *ResourcePermissions) DoCreate(ctx context.Context, newState *PermissionsState) (string, error) {
	// should we remember the default here?
	err := r.DoUpdate(ctx, newState.ObjectID, newState)
	if err != nil {
		return "", err
	}

	return newState.ObjectID, nil
}

// DoUpdate calls https://docs.databricks.com/api/workspace/jobs/setjobpermissions.
func (r *ResourcePermissions) DoUpdate(ctx context.Context, _ string, newState *PermissionsState) error {
	extractedType, extractedID, err := parsePermissionsID(newState.ObjectID)
	if err != nil {
		return err
	}

	_, err = r.client.Permissions.Set(ctx, iam.SetObjectPermissions{
		RequestObjectId:   extractedID,
		RequestObjectType: extractedType,
		AccessControlList: newState.Permissions,
	})

	return err
}

// DoDelete is activated in 2 distinct cases:
// 1) 'permissions' field is deleted in DABs config. In that case terraform would restore the default permissions (IS_OWNER for current user).
// 2) the parent resource is deleted; in that case there is no need to do anything; parent resource deletion is enough.
// Let's do nothing in both cases. If user no longer wishes to manage permissions with DABs they can go ahead and manage
// it themselves. Trying to fix permissions back requires
// - making assumptions on what it should look like
// - storing current user somewhere or storing original permissions somewhere
func (r *ResourcePermissions) DoDelete(ctx context.Context, id string) error {
	// high performance delete implementation:
	return nil
}
