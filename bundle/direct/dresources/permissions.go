package dresources

import (
	"context"
	"fmt"
	"slices"
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

	slices.SortStableFunc(result.Permissions, sortKey)
	return &result, nil
}

func sortKey(a, b iam.AccessControlRequest) int {
	// First order by field userd: UserName first, then GroupName then ServicePrincipalName
	result := getOrder(a) - getOrder(b)
	if result != 0 {
		return result
	}
	if a.UserName != "" {
		return strings.Compare(a.UserName, b.UserName)
	}
	if b.GroupName != "" {
		return strings.Compare(a.GroupName, b.GroupName)
	}
	return strings.Compare(a.ServicePrincipalName, b.ServicePrincipalName)
}

func getOrder(a iam.AccessControlRequest) int {
	if a.UserName != "" {
		return 1
	}
	if a.GroupName != "" {
		return 2
	}
	// a.ServicePrincipalName != ""
	return 3
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

	// Note, this sorts in place and is reflected in new_state. The purpose here is to ensure we're resilient against backend randomising order.
	// The downside is that we create a different order from what is visible to use in the config. Proper solution would be to keep
	// user's order and then adapt remote state order to user's order. This is the same issues as with job task_key, so there maybe a common implementation.
	slices.SortStableFunc(newState.Permissions, sortKey)

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
