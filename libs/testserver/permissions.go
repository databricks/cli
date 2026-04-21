package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/iam"
)

// source: https://github.com/databricks/terraform-provider-databricks/blob/main/permissions/permission_definitions.go

var requestObjectTypeToObjectType = map[string]string{
	"cluster-policies":        "cluster-policy",
	"instance-pools":          "instance-pool",
	"clusters":                "cluster",
	"pipelines":               "pipelines",
	"jobs":                    "job",
	"notebooks":               "notebook",
	"directories":             "directory",
	"files":                   "file",
	"repos":                   "repo",
	"authorization":           "tokens", // maps to both "tokens" and "passwords"
	"sql/warehouses":          "warehouses",
	"dbsql-dashboards":        "dashboard",
	"sql/alerts":              "alert",
	"sql/queries":             "query",
	"dashboards":              "dashboard",
	"experiments":             "mlflowExperiment",
	"registered-models":       "registered-model",
	"serving-endpoints":       "serving-endpoint",
	"vector-search-endpoints": "vector-search-endpoints",
	"apps":                    "apps",
	"app-spaces":              "app-spaces",
	"database-instances":      "database-instances",
	"database-projects":       "database-projects",
	"alertsv2":                "alertv2",
}

// aclPrincipalKey returns a unique key identifying the principal in an ACL entry.
func aclPrincipalKey(acl iam.AccessControlResponse) string {
	switch {
	case acl.UserName != "":
		return "user:" + acl.UserName
	case acl.GroupName != "":
		return "group:" + acl.GroupName
	case acl.ServicePrincipalName != "":
		return "sp:" + acl.ServicePrincipalName
	default:
		return ""
	}
}

// upsertACL adds or replaces an ACL entry by principal key.
// Entries with no principal key are ignored.
func upsertACL(perms *iam.ObjectPermissions, entry iam.AccessControlResponse) {
	key := aclPrincipalKey(entry)
	if key == "" {
		return
	}
	for i, acl := range perms.AccessControlList {
		if aclPrincipalKey(acl) == key {
			perms.AccessControlList[i] = entry
			return
		}
	}
	perms.AccessControlList = append(perms.AccessControlList, entry)
}

// upsertPermission adds or replaces an ACL entry on the given object.
// Must be called with s.mu held.
func (s *FakeWorkspace) upsertPermission(objectKey string, entry iam.AccessControlResponse) {
	perms := s.Permissions[objectKey]
	upsertACL(&perms, entry)
	s.Permissions[objectKey] = perms
}

// GetPermissions retrieves permissions for a given object type and ID
func (s *FakeWorkspace) GetPermissions(req Request) any {
	defer s.LockUnlock()()

	objectId := req.Vars["object_id"]
	requestObjectType := req.Vars["object_type"]
	prefix := req.Vars["prefix"]
	if prefix != "" {
		requestObjectType = prefix + "/" + requestObjectType
	}

	if requestObjectType == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "object_type is required"},
		}
	}

	objectType := requestObjectTypeToObjectType[requestObjectType]
	if objectType == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "request_object_type is not recognized: " + requestObjectType},
		}
	}

	if objectId == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "object_id is required"},
		}
	}

	responseObjectID := fmt.Sprintf("/%s/%s", requestObjectType, objectId)
	permissions, exists := s.Permissions[responseObjectID]

	if !exists {
		// Return empty permissions structure if not found
		permissions = iam.ObjectPermissions{
			ObjectId:          responseObjectID,
			ObjectType:        objectType,
			AccessControlList: []iam.AccessControlResponse{},
		}
	}

	return Response{
		Body: permissions,
	}
}

func (s *FakeWorkspace) SetPermissions(req Request) any {
	defer s.LockUnlock()()

	objectId := req.Vars["object_id"]
	requestObjectType := req.Vars["object_type"]
	prefix := req.Vars["prefix"]
	if prefix != "" {
		requestObjectType = prefix + "/" + requestObjectType
	}

	if requestObjectType == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "request_object_type is required"},
		}
	}

	objectType := requestObjectTypeToObjectType[requestObjectType]
	if objectType == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "request_object_type is not recognized: " + requestObjectType},
		}
	}

	if objectId == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "object_id is required"},
		}
	}

	var updateRequest iam.UpdateObjectPermissions
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse permissions update: %s", err)},
		}
	}

	responseObjectID := fmt.Sprintf("/%s/%s", requestObjectType, objectId)

	// Get existing permissions or create new ones
	existingPermissions, exists := s.Permissions[responseObjectID]
	if !exists {
		existingPermissions = iam.ObjectPermissions{
			ObjectId:          responseObjectID,
			ObjectType:        objectType,
			AccessControlList: []iam.AccessControlResponse{},
		}
	}

	// Convert AccessControlRequest to AccessControlResponse and replace the ACL.
	existingPermissions.AccessControlList = nil
	for _, acl := range updateRequest.AccessControlList {
		display := acl.UserName
		if display == "" {
			display = acl.ServicePrincipalName
		}

		response := iam.AccessControlResponse{
			UserName:             acl.UserName,
			GroupName:            acl.GroupName,
			ServicePrincipalName: acl.ServicePrincipalName,
			DisplayName:          display,
		}

		if acl.PermissionLevel != "" {
			response.AllPermissions = []iam.Permission{{
				Inherited:       false,
				PermissionLevel: acl.PermissionLevel,
				ForceSendFields: []string{"Inherited"},
			}}
		}

		upsertACL(&existingPermissions, response)
	}

	// Apply cloud environment fixups - better match cloud env
	if requestObjectType == "jobs" {
		existingPermissions.AccessControlList = append(existingPermissions.AccessControlList, iam.AccessControlResponse{
			AllPermissions: []iam.Permission{
				{
					Inherited:           true,
					InheritedFromObject: []string{"/jobs/"},
					PermissionLevel:     "CAN_MANAGE",
				},
			},
			GroupName: "admins",
		})
	}

	// Add default ACLs for alertsv2 to match cloud environment
	if requestObjectType == "alertsv2" {
		existingPermissions.AccessControlList = append(existingPermissions.AccessControlList, iam.AccessControlResponse{
			AllPermissions: []iam.Permission{
				{
					Inherited:           true,
					InheritedFromObject: []string{"/directories/4454031293888593"},
					PermissionLevel:     "CAN_MANAGE",
				},
			},
			UserName:    "shreyas.goenka@databricks.com",
			DisplayName: "shreyas.goenka@databricks.com",
		}, iam.AccessControlResponse{
			AllPermissions: []iam.Permission{
				{
					Inherited:           true,
					InheritedFromObject: []string{"/directories/"},
					PermissionLevel:     "CAN_MANAGE",
				},
			},
			GroupName: "admins",
		})
	}

	// Validate job ownership requirements
	if requestObjectType == "jobs" {
		hasOwner := false
		for _, acl := range existingPermissions.AccessControlList {
			for _, perm := range acl.AllPermissions {
				if perm.PermissionLevel == "IS_OWNER" {
					hasOwner = true
					break
				}
			}
			if hasOwner {
				break
			}
		}

		if !hasOwner {
			return Response{
				StatusCode: 400,
				Body:       map[string]string{"message": "The job must have exactly one owner."},
			}
		}
	}

	s.Permissions[responseObjectID] = existingPermissions

	return Response{
		Body: existingPermissions,
	}
}
