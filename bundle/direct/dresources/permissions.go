package dresources

import (
	"context"
	"fmt"
	"reflect"
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

	// Use reflection to get the slice from the pointer
	rv := reflect.ValueOf(inputConfig)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return nil, fmt.Errorf("inputConfig must be a pointer to a slice, got: %T", inputConfig)
	}

	sliceValue := rv.Elem()

	// Get the element type from the slice type and create zero value to get the object type
	elemType := sliceValue.Type().Elem()
	zeroValue := reflect.Zero(elemType)
	zeroValueInterface, ok := zeroValue.Interface().(resources.IPermission)
	if !ok {
		return nil, fmt.Errorf("slice elements do not implement IPermission interface: %v", elemType)
	}
	prefix := zeroValueInterface.GetAPIRequestObjectType()

	// Convert slice to []resources.IPermission
	permissions := make([]iam.AccessControlRequest, 0, sliceValue.Len())
	for i := range sliceValue.Len() {
		elem := sliceValue.Index(i).Interface().(resources.IPermission)
		permissions = append(permissions, iam.AccessControlRequest{
			PermissionLevel:      iam.PermissionLevel(elem.GetLevel()),
			GroupName:            elem.GetGroupName(),
			ServicePrincipalName: elem.GetServicePrincipalName(),
			UserName:             elem.GetUserName(),
			ForceSendFields:      nil,
		})
	}

	objectIdRef := prefix + "${" + baseNode + ".id}"
	// For permissions, model serving endpoint uses it's internal ID, which is different
	// from its CRUD APIs which use the name.
	// We have a wrapper struct [RefreshOutput] from which we read the internal ID
	// in order to set the appropriate permissions.
	if strings.HasPrefix(baseNode, "resources.model_serving_endpoints.") {
		objectIdRef = prefix + "${" + baseNode + ".endpoint_id}"
	}

	return &structvar.StructVar{
		Value: &PermissionsState{
			ObjectID:    "", // Always a reference, defined in Refs below
			Permissions: permissions,
		},
		Refs: map[string]string{
			"object_id": objectIdRef,
		},
	}, nil
}

func (*ResourcePermissions) New(client *databricks.WorkspaceClient) *ResourcePermissions {
	return &ResourcePermissions{client: client}
}

func (*ResourcePermissions) PrepareState(s *PermissionsState) *PermissionsState {
	return s
}

func accessControlRequestKey(x iam.AccessControlRequest) (string, string) {
	if x.UserName != "" {
		return "user_name", x.UserName
	}
	if x.ServicePrincipalName != "" {
		return "service_principal_name", x.ServicePrincipalName
	}
	if x.GroupName != "" {
		return "group_name", x.GroupName
	}
	return "", ""
}

func (*ResourcePermissions) KeyedSlices() map[string]any {
	return map[string]any{
		"permissions": accessControlRequestKey,
	}
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

func (r *ResourcePermissions) DoRead(ctx context.Context, id string) (*PermissionsState, error) {
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
func (r *ResourcePermissions) DoCreate(ctx context.Context, newState *PermissionsState) (string, *PermissionsState, error) {
	// should we remember the default here?
	_, err := r.DoUpdate(ctx, newState.ObjectID, newState, nil)
	if err != nil {
		return "", nil, err
	}

	return newState.ObjectID, nil, nil
}

// DoUpdate calls https://docs.databricks.com/api/workspace/jobs/setjobpermissions.
func (r *ResourcePermissions) DoUpdate(ctx context.Context, _ string, newState *PermissionsState, _ *Changes) (*PermissionsState, error) {
	extractedType, extractedID, err := parsePermissionsID(newState.ObjectID)
	if err != nil {
		return nil, err
	}

	_, err = r.client.Permissions.Set(ctx, iam.SetObjectPermissions{
		RequestObjectId:   extractedID,
		RequestObjectType: extractedType,
		AccessControlList: newState.Permissions,
	})

	return nil, err
}

// DoDelete is activated in 2 distinct cases:
// 1) 'permissions' field is deleted in DABs config. In that case terraform would restore the default permissions (IS_OWNER for current user).
// 2) the parent resource is deleted; in that case there is no need to do anything; parent resource deletion is enough.
// Let's do nothing in both cases. If user no longer wishes to manage permissions with DABs they can go ahead and manage
// it themselves. Trying to fix permissions back requires
// - making assumptions on what it should look like
// - storing current user somewhere or storing original permissions somewhere
func (r *ResourcePermissions) DoDelete(ctx context.Context, id string) error {
	// intentional noop
	return nil
}
