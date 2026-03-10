package dresources

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// GetAPIRequestObjectType is used by direct to construct a request to permissions API:
// https://github.com/databricks/terraform-provider-databricks/blob/430902d/permissions/permission_definitions.go#L775C24-L775C32
var permissionResourceToObjectType = map[string]string{
	"alerts":                  "/alertsv2/",
	"apps":                    "/apps/",
	"clusters":                "/clusters/",
	"dashboards":              "/dashboards/",
	"database_instances":      "/database-instances/",
	"postgres_projects":       "/database-projects/",
	"jobs":                    "/jobs/",
	"experiments":             "/experiments/",
	"models":                  "/registered-models/",
	"model_serving_endpoints": "/serving-endpoints/",
	"pipelines":               "/pipelines/",
	"sql_warehouses":          "/sql/warehouses/",
}

type ResourcePermissions struct {
	client *databricks.WorkspaceClient
}

type PermissionsState struct {
	ObjectID      string                     `json:"object_id"`
	EmbeddedSlice []iam.AccessControlRequest `json:"permissions,omitempty"`
}

func PreparePermissionsInputConfig(inputConfig any, node string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(node, ".permissions")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .permissions", node)
	}

	parts := strings.Split(baseNode, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("internal error: unexpected node format %q", baseNode)
	}
	resourceType := parts[1]

	prefix, ok := permissionResourceToObjectType[resourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported permissions resource type: %s", resourceType)
	}

	permissions, err := toAccessControlRequests(inputConfig)
	if err != nil {
		return nil, err
	}

	objectIdRef := prefix + "${" + baseNode + ".id}"
	// For permissions, model serving endpoint uses its internal ID, which is different
	// from its CRUD APIs which use the name.
	// We have a wrapper struct [RefreshOutput] from which we read the internal ID
	// in order to set the appropriate permissions.
	if strings.HasPrefix(baseNode, "resources.model_serving_endpoints.") {
		objectIdRef = prefix + "${" + baseNode + ".endpoint_id}"
	}

	// Postgres projects store their hierarchical name ("projects/{project_id}") as the state ID,
	// but the permissions API expects just the project_id.
	if strings.HasPrefix(baseNode, "resources.postgres_projects.") {
		objectIdRef = prefix + "${" + baseNode + ".project_id}"
	}

	return &structvar.StructVar{
		Value: &PermissionsState{
			ObjectID:    "", // Always a reference, defined in Refs below
			EmbeddedSlice: permissions,
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

// toAccessControlRequests converts any slice of permission structs to []iam.AccessControlRequest.
// All permission types share the same underlying struct layout (Level, UserName, ServicePrincipalName, GroupName).
func toAccessControlRequests(ps any) ([]iam.AccessControlRequest, error) {
	v := reflect.ValueOf(ps)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected permissions slice, got %T", ps)
	}
	result := make([]iam.AccessControlRequest, v.Len())
	for i := range v.Len() {
		elem := v.Index(i)
		result[i] = iam.AccessControlRequest{
			PermissionLevel:      iam.PermissionLevel(elem.FieldByName("Level").String()),
			UserName:             elem.FieldByName("UserName").String(),
			ServicePrincipalName: elem.FieldByName("ServicePrincipalName").String(),
			GroupName:            elem.FieldByName("GroupName").String(),
			ForceSendFields:      nil,
		}
	}
	return result, nil
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
	// Empty key because EmbeddedSlice appears at the root path of
	// PermissionsState (no "permissions" prefix in struct walker paths).
	return map[string]any{
		"": accessControlRequestKey,
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
		ObjectID:      id,
		EmbeddedSlice: nil,
	}

	for _, accessControl := range acl.AccessControlList {
		for _, permission := range accessControl.AllPermissions {
			// Inherited permissions can be ignored, as they are not set by the user (following TF)
			if permission.Inherited {
				continue
			}
			result.EmbeddedSlice = append(result.EmbeddedSlice, iam.AccessControlRequest{
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
func (r *ResourcePermissions) DoUpdate(ctx context.Context, _ string, newState *PermissionsState, _ Changes) (*PermissionsState, error) {
	extractedType, extractedID, err := parsePermissionsID(newState.ObjectID)
	if err != nil {
		return nil, err
	}

	_, err = r.client.Permissions.Set(ctx, iam.SetObjectPermissions{
		RequestObjectId:   extractedID,
		RequestObjectType: extractedType,
		AccessControlList: newState.EmbeddedSlice,
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
