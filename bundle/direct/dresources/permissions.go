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
	"vector_search_endpoints": "/vector-search-endpoints/",
}

type ResourcePermissions struct {
	client *databricks.WorkspaceClient
}

// StatePermission represents a permission entry in deployment state.
type StatePermission struct {
	Level                iam.PermissionLevel `json:"level,omitempty"`
	UserName             string              `json:"user_name,omitempty"`
	ServicePrincipalName string              `json:"service_principal_name,omitempty"`
	GroupName            string              `json:"group_name,omitempty"`
}

// Note __embed__ name is not enforced by libs/structs, it's a convention we follow here.
// Technically we could keep on using "permissions" or "grants" for this, but this would make it
// harder for 3rd-party tools that only see JSON and not Golang type to associate state object with the pass from "changes".
// Alternative to __embed_ would be have a convention that we name this inner field same as outer field.
// e.g. permissions.permissions or grants.grants. However, there is no guarantee that this convention is not also triggered
// by unrelated types and it's harder to evaluate that fixed string because it's non-local.

type PermissionsState struct {
	ObjectID string `json:"object_id"`
	// By convention, EmbedSlice fields should have __embed__ json tag, see permissions.go for details
	EmbeddedSlice []StatePermission `json:"__embed__,omitempty"`
}

// permissionIDFields maps resource types that use a non-standard ID field for
// the permissions API (most resources use "id").
var permissionIDFields = map[string]string{
	"model_serving_endpoints": "endpoint_id",   // internal numeric ID, not the name used in CRUD APIs
	"models":                  "model_id",      // numeric model ID, not the model name used as CRUD state ID
	"postgres_projects":       "project_id",    // bare project_id, not the hierarchical "projects/{id}" state ID
	"vector_search_endpoints": "endpoint_uuid", // endpoint UUID, not the endpoint name used as deployment ID
}

// objectIDRef returns the reference expression for the permissions object ID.
func objectIDRef(prefix, baseNode, resourceType string) string {
	if field, ok := permissionIDFields[resourceType]; ok {
		return prefix + "${" + baseNode + "." + field + "}"
	}
	return prefix + "${" + baseNode + ".id}"
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

	permissions, err := toStatePermissions(inputConfig)
	if err != nil {
		return nil, err
	}

	return &structvar.StructVar{
		Value: &PermissionsState{
			ObjectID:      "", // Always a reference, defined in Refs below
			EmbeddedSlice: permissions,
		},
		Refs: map[string]string{
			"object_id": objectIDRef(prefix, baseNode, resourceType),
		},
	}, nil
}

func (*ResourcePermissions) New(client *databricks.WorkspaceClient) *ResourcePermissions {
	return &ResourcePermissions{client: client}
}

func (*ResourcePermissions) PrepareState(s *PermissionsState) *PermissionsState {
	return s
}

// toStatePermissions converts any slice of typed permission structs (e.g. []JobPermission)
// to []StatePermission. All permission types share the same underlying struct layout.
func toStatePermissions(ps any) ([]StatePermission, error) {
	v := reflect.ValueOf(ps)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected permissions slice, got %T", ps)
	}
	result := make([]StatePermission, v.Len())
	for i := range v.Len() {
		elem := v.Index(i)
		result[i] = StatePermission{
			Level:                iam.PermissionLevel(elem.FieldByName("Level").String()),
			UserName:             elem.FieldByName("UserName").String(),
			ServicePrincipalName: elem.FieldByName("ServicePrincipalName").String(),
			GroupName:            elem.FieldByName("GroupName").String(),
		}
	}
	return result, nil
}

func permissionKey(x StatePermission) (string, string) {
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
		"": permissionKey,
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
			result.EmbeddedSlice = append(result.EmbeddedSlice, StatePermission{
				Level:                permission.PermissionLevel,
				GroupName:            accessControl.GroupName,
				UserName:             accessControl.UserName,
				ServicePrincipalName: accessControl.ServicePrincipalName,
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
func (r *ResourcePermissions) DoUpdate(ctx context.Context, _ string, newState *PermissionsState, _ *PlanEntry) (*PermissionsState, error) {
	extractedType, extractedID, err := parsePermissionsID(newState.ObjectID)
	if err != nil {
		return nil, err
	}

	acl := make([]iam.AccessControlRequest, len(newState.EmbeddedSlice))
	for i, p := range newState.EmbeddedSlice {
		acl[i] = iam.AccessControlRequest{
			PermissionLevel:      p.Level,
			UserName:             p.UserName,
			ServicePrincipalName: p.ServicePrincipalName,
			GroupName:            p.GroupName,
			ForceSendFields:      nil,
		}
	}

	_, err = r.client.Permissions.Set(ctx, iam.SetObjectPermissions{
		RequestObjectId:   extractedID,
		RequestObjectType: extractedType,
		AccessControlList: acl,
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
