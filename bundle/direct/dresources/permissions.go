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

type PermissionsInput[T any] struct {
	ObjectID    string `json:"object_id"`
	Permissions []T    `json:"permissions,omitempty"`
}

func PreparePermissionsInputConfig(inputConfig any, node string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(node, ".permissions")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .permissions", node)
	}
	switch v := inputConfig.(type) {
	case *[]resources.JobPermission:
		return &structvar.StructVar{
			Config: &PermissionsInput[resources.JobPermission]{
				ObjectID:    "", // Always a reference, defined in Refs below
				Permissions: *v,
			},
			Refs: map[string]string{
				"object_id": "/jobs/${" + baseNode + ".id}",
			},
		}, nil
	case *[]resources.PipelinePermission:
		return &structvar.StructVar{
			Config: &PermissionsInput[resources.PipelinePermission]{
				ObjectID:    "",
				Permissions: *v,
			},
			Refs: map[string]string{
				"object_id": "/pipelines/${" + baseNode + ".id}",
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported type for permissions: %T", inputConfig)
	}
}

type PermissionsState struct {
	ObjectID    string                     `json:"object_id"`
	Permissions []iam.AccessControlRequest `json:"permissions,omitempty"`
}

func (*ResourcePermissions) New(client *databricks.WorkspaceClient) *ResourcePermissions {
	return &ResourcePermissions{client: client}
}

func (*ResourcePermissions) PrepareState(input any) *PermissionsState {
	switch v := input.(type) {
	case *PermissionsInput[resources.JobPermission]:
		result := PermissionsState{
			// Note PermissionsInput is a StructVar, so it consists of Config and Refs.
			// We only receive Config there, refs are copied directly in bundle/direct/bundle_plan.go
			// ObjectID is a reference in bundle_plan.go but all_test.go passes concrete value
			ObjectID:    v.ObjectID,
			Permissions: nil,
		}
		for _, p := range v.Permissions {
			result.Permissions = append(result.Permissions, iam.AccessControlRequest{
				GroupName:            p.GroupName,
				PermissionLevel:      iam.PermissionLevel(p.Level),
				ServicePrincipalName: p.ServicePrincipalName,
				UserName:             p.UserName,
				ForceSendFields:      nil,
			})
		}
		return &result
	case *PermissionsInput[resources.PipelinePermission]:
		result := PermissionsState{
			// Note PermissionsInput is a StructVar, so it consists of Config and Refs.
			// We only receive Config there, refs are copied directly in bundle/direct/bundle_plan.go
			// ObjectID is a reference in bundle_plan.go but all_test.go passes concrete value
			ObjectID:    v.ObjectID,
			Permissions: nil,
		}
		for _, p := range v.Permissions {
			result.Permissions = append(result.Permissions, iam.AccessControlRequest{
				GroupName:            p.GroupName,
				PermissionLevel:      iam.PermissionLevel(p.Level),
				ServicePrincipalName: p.ServicePrincipalName,
				UserName:             p.UserName,
				ForceSendFields:      nil,
			})
		}
		return &result
	default:
		return nil
	}
}

/*
// GET returns
type ObjectPermissions struct {
	AccessControlList []AccessControlResponse `json:"access_control_list,omitempty"`

	ObjectId string `json:"object_id,omitempty"`

	ObjectType string `json:"object_type,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

type AccessControlResponse struct {
	// All permissions.
	AllPermissions []Permission `json:"all_permissions,omitempty"`
	// Display name of the user or service principal.
	DisplayName string `json:"display_name,omitempty"`
	// name of the group
	GroupName string `json:"group_name,omitempty"`
	// Name of the service principal.
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	// name of the user
	UserName string `json:"user_name,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

type Permission struct {
	Inherited bool `json:"inherited,omitempty"`

	InheritedFromObject []string `json:"inherited_from_object,omitempty"`

	PermissionLevel PermissionLevel `json:"permission_level,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}





*/

func (r *ResourcePermissions) DoRefresh(ctx context.Context, id string) (*PermissionsState, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) != 3 { // "/jobs/123"
		return nil, fmt.Errorf("cannot parse id: %q", id)
	}

	extractedType := idParts[1]
	extractedID := idParts[2]

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
	idParts := strings.Split(newState.ObjectID, "/")
	if len(idParts) != 3 { // "/jobs/123"
		return fmt.Errorf("cannot parse id: %q", newState.ObjectID)
	}

	extractedType := idParts[1]
	extractedID := idParts[2]

	_, err := r.client.Permissions.Set(ctx, iam.SetObjectPermissions{
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
