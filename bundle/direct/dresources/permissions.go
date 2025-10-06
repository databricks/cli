package dresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/tfpermissions"
	"github.com/databricks/cli/libs/tfpermissions/entity"
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
	permConfig, err := tfpermissions.GetResourcePermissionsFromId(id)
	if err != nil {
		return nil, err
	}

	// TODO: handle this
	existing := entity.PermissionsEntity{
		ObjectType:        "",
		AccessControlList: nil,
	}
	me := " this must never match existing user !@#$%"

	response, err := tfpermissions.NewPermissionsAPI(r.client).Read(ctx, id, permConfig, existing, me)
	if err != nil {
		return nil, err
	}

	return &PermissionsState{
		ObjectID:    id,
		Permissions: response.AccessControlList,
	}, nil
}

// DoCreate calls https://docs.databricks.com/api/workspace/jobs/setjobpermissions.
func (r *ResourcePermissions) DoCreate(ctx context.Context, newState *PermissionsState) (string, error) {
	err := r.DoUpdate(ctx, newState.ObjectID, newState)
	if err != nil {
		return "", err
	}

	return newState.ObjectID, nil
}

// DoUpdate calls https://docs.databricks.com/api/workspace/jobs/setjobpermissions.
func (r *ResourcePermissions) DoUpdate(ctx context.Context, id string, newState *PermissionsState) error {
	permConfig, err := tfpermissions.GetResourcePermissionsFromId(id)
	if err != nil {
		return fmt.Errorf("getting permissions config for %q: %w", id, err)
	}

	entity := entity.PermissionsEntity{
		ObjectType:        permConfig.ObjectType(),
		AccessControlList: newState.Permissions,
	}

	return tfpermissions.NewPermissionsAPI(r.client).Update(ctx, id, entity, permConfig)
}

// DoDelete clears ACLs through https://docs.databricks.com/api/workspace/jobs/setjobpermissions.
func (r *ResourcePermissions) DoDelete(ctx context.Context, id string) error {
	return r.DoUpdate(ctx, id, &PermissionsState{
		ObjectID:    id,
		Permissions: nil,
	})
}
