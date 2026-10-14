package testserver

import (
	"encoding/json"
	"fmt"
	"github.com/databricks/cli/libs/utils"
	"slices"
	"strconv"
	"strings"

	iam "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
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
	"genie":                   "genie-space",
	"experiments":             "mlflowExperiment",
	"registered-models":       "registered-model",
	"serving-endpoints":       "serving-endpoint",
	"vector-search-endpoints": "vector-search-endpoints",
	"apps":                    "apps",
	"database-instances":      "database-instances",
	"database-projects":       "database-projects",
	"alertsv2":                "alertv2",
}

// aclPrincipalKey returns a unique key identifying the principal in an ACL entry.
func aclPrincipalKey(acl iam.AccessControlResponse) string {
	switch {
	case utils.Deref(acl.UserName) != "":
		return "user:" + *acl.UserName
	case utils.Deref(acl.GroupName) != "":
		return "group:" + *acl.GroupName
	case utils.Deref(acl.ServicePrincipalName) != "":
		return "sp:" + *acl.ServicePrincipalName
	default:
		return ""
	}
}

// upsertACL adds or replaces an ACL entry by principal key.
// Entries with no principal key are ignored.
func upsertACL(perms *iam.PermissionsResponse, entry iam.AccessControlResponse) {
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

// jobPrincipalHasAccess reports whether the principal has a direct ACL entry
// (hasAccess) and whether the ACL names an explicit owner (hasOwner). hasOwner
// lets callers grant the creator implicit access only until ownership moves.
func jobPrincipalHasAccess(perms iam.PermissionsResponse, principal string) (hasAccess, hasOwner bool) {
	for _, acl := range perms.AccessControlList {
		for _, p := range acl.AllPermissions {
			if p.PermissionLevel == "IS_OWNER" {
				hasOwner = true
			}
		}
		if len(acl.AllPermissions) > 0 && (utils.Deref(acl.UserName) == principal || utils.Deref(acl.ServicePrincipalName) == principal) {
			hasAccess = true
		}
	}
	return hasAccess, hasOwner
}

// guestHasJobAccess reports whether a guest may access a job: via a direct ACL
// entry, or as the creator while no explicit owner is set. Call with s.mu held.
func (s *FakeWorkspace) guestHasJobAccess(jobId int64, principal string) bool {
	perms := s.Permissions[fmt.Sprintf("/jobs/%d", jobId)]
	hasAccess, hasOwner := jobPrincipalHasAccess(perms, principal)
	if hasAccess {
		return true
	}
	if !hasOwner {
		if job, ok := s.Jobs[jobId]; ok && utils.Deref(job.CreatorUserName) == principal {
			return true
		}
	}
	return false
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

	// A guest without manage access cannot read a job's permissions, mirroring
	// the backend's pre-existence permission check.
	if requestObjectType == "jobs" && isGuestToken(req.Token) {
		if jobId, err := strconv.ParseInt(objectId, 10, 64); err == nil && !s.guestHasJobAccess(jobId, utils.Deref(userForToken(req.Token).UserName)) {
			return jobManagePermissionDenied(utils.Deref(userForToken(req.Token).UserName), jobId)
		}
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

	// V2 permissions APIs cascade-delete ACLs with the parent, so the cloud
	// returns 404 once the parent is gone. V1 APIs (jobs, pipelines, etc.)
	// retain ACL data after delete via async/soft delete; for those, we
	// fall through to the "empty ACL on miss" branch below, which is close
	// enough. New V2 resources should add a case to permissionsParentExists.
	if !s.permissionsParentExists(requestObjectType, objectId) {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("%s %s not found.", requestObjectType, objectId)},
		}
	}

	permissions, exists := s.Permissions[responseObjectID]

	if !exists {
		// Return empty permissions structure if not found
		permissions = iam.PermissionsResponse{
			ObjectId:          utils.NonZero(responseObjectID),
			ObjectType:        utils.NonZero(objectType),
			AccessControlList: []iam.AccessControlResponse{},
		}
	}

	return Response{
		Body: permissions,
	}
}

// permissionsParentExists reports whether the parent object backing a
// permissions request exists in workspace state. Returns true for resource
// types without a parent-existence check wired up; V1 resources rely on
// that fallback to keep their "empty ACL on miss" behavior.
func (s *FakeWorkspace) permissionsParentExists(requestObjectType, objectId string) bool {
	switch requestObjectType {
	case "vector-search-endpoints":
		for _, ep := range s.VectorSearchEndpoints {
			if utils.Deref(ep.Id) == objectId {
				return true
			}
		}
		return false
	}
	return true
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

	var updateRequest iam.UpdateObjectPermissionsRequest
	if err := json.Unmarshal(req.Body, &updateRequest); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse permissions update: %s", err)},
		}
	}

	// Cluster policies only support CAN_USE; the real API rejects other levels
	// with this exact message. Model it so deploys fail the same way on cloud.
	if requestObjectType == "cluster-policies" {
		for _, acl := range updateRequest.AccessControlList {
			if acl.PermissionLevel != "" && acl.PermissionLevel != "CAN_USE" {
				return Response{
					StatusCode: 400,
					Body: map[string]string{
						"error_code": "INVALID_PARAMETER_VALUE",
						"message":    fmt.Sprintf("Unknown Cluster Policy Permission Level: %s", acl.PermissionLevel),
					},
				}
			}
		}
	}

	responseObjectID := fmt.Sprintf("/%s/%s", requestObjectType, objectId)

	// Get existing permissions or create new ones
	existingPermissions, exists := s.Permissions[responseObjectID]
	if !exists {
		existingPermissions = iam.PermissionsResponse{
			ObjectId:          utils.NonZero(responseObjectID),
			ObjectType:        utils.NonZero(objectType),
			AccessControlList: []iam.AccessControlResponse{},
		}
	}

	// Convert AccessControlRequest to AccessControlResponse and replace the ACL.
	existingPermissions.AccessControlList = nil
	for _, acl := range updateRequest.AccessControlList {
		display := acl.UserName
		if utils.Deref(display) == "" {
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
				Inherited:       new(false),
				PermissionLevel: acl.PermissionLevel,
			}}
		}

		upsertACL(&existingPermissions, response)
	}

	// Apply cloud environment fixups - better match cloud env
	if requestObjectType == "jobs" {
		existingPermissions.AccessControlList = append(existingPermissions.AccessControlList, iam.AccessControlResponse{
			AllPermissions: []iam.Permission{
				{
					Inherited:           utils.NonZero(true),
					InheritedFromObject: []string{"/jobs/"},
					PermissionLevel:     "CAN_MANAGE",
				},
			},
			GroupName: utils.NonZero("admins"),
		})
	}

	// Add default ACLs for alertsv2 to match cloud environment
	if requestObjectType == "alertsv2" {
		existingPermissions.AccessControlList = append(existingPermissions.AccessControlList, iam.AccessControlResponse{
			AllPermissions: []iam.Permission{
				{
					Inherited:           utils.NonZero(true),
					InheritedFromObject: []string{"/directories/4454031293888593"},
					PermissionLevel:     "CAN_MANAGE",
				},
			},
			UserName:    utils.NonZero("shreyas.goenka@databricks.com"),
			DisplayName: utils.NonZero("shreyas.goenka@databricks.com"),
		}, iam.AccessControlResponse{
			AllPermissions: []iam.Permission{
				{
					Inherited:           utils.NonZero(true),
					InheritedFromObject: []string{"/directories/"},
					PermissionLevel:     "CAN_MANAGE",
				},
			},
			GroupName: utils.NonZero("admins"),
		})
	}

	// Jobs and pipelines require exactly one owner. The real Permissions API rejects
	// a PUT with zero owners OR more than one owner (verified against the backend);
	// both cases return "The <type> must have exactly one owner."
	ownerNoun := map[string]string{"jobs": "job", "pipelines": "pipeline"}[requestObjectType]
	if ownerNoun != "" {
		owners := 0
		for _, acl := range existingPermissions.AccessControlList {
			for _, perm := range acl.AllPermissions {
				if perm.PermissionLevel == "IS_OWNER" {
					owners++
				}
			}
		}

		if owners != 1 {
			return Response{
				StatusCode: 400,
				Body: map[string]string{
					"error_code": "INVALID_PARAMETER_VALUE",
					"message":    "The " + ownerNoun + " must have exactly one owner.",
				},
			}
		}
	}

	s.Permissions[responseObjectID] = existingPermissions

	// A directory under /Workspace/Users/<owner> grants the owner CAN_MANAGE
	// (inherited), mirroring real workspaces where a user manages everything in their
	// home folder. This holds regardless of what the request configured for them, so
	// it overrides any lower level. SetPermissions replaces the direct ACL but this
	// inherited access persists, so we add it to the response (not the stored value).
	response := existingPermissions
	if requestObjectType == "directories" {
		if owner := s.directoryHomeOwner(objectId); owner != "" {
			response.AccessControlList = slices.Clone(existingPermissions.AccessControlList)
			upsertACL(&response, iam.AccessControlResponse{
				UserName:       utils.NonZero(owner),
				AllPermissions: []iam.Permission{{PermissionLevel: "CAN_MANAGE", Inherited: utils.NonZero(true)}},
			})
		}
	}

	return Response{
		Body: response,
	}
}

// directoryHomeOwner returns the home-directory owner for the directory with the
// given object id, i.e. <owner> when its path is under /Workspace/Users/<owner>.
// Returns an empty string otherwise.
func (s *FakeWorkspace) directoryHomeOwner(objectId string) string {
	const usersPrefix = "/Workspace/Users/"
	for path, info := range s.directories {
		if strconv.FormatInt(info.ObjectId, 10) != objectId {
			continue
		}
		if !strings.HasPrefix(path, usersPrefix) {
			return ""
		}
		rest := path[len(usersPrefix):]
		if before, _, ok := strings.Cut(rest, "/"); ok {
			return before
		}
		return rest
	}
	return ""
}
