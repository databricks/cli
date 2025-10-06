package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/iam"
)

// GetPermissions retrieves permissions for a given object type and ID
func (s *FakeWorkspace) GetPermissions(req Request) any {
	defer s.LockUnlock()()

	objectType := req.Vars["object_type"]
	objectId := req.Vars["object_id"]

	if objectType == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "object_type is required"},
		}
	}

	if objectId == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "object_id is required"},
		}
	}

	permissionKey := fmt.Sprintf("%s:%s", objectType, objectId)
	permissions, exists := s.Permissions[permissionKey]

	if !exists {
		// Return empty permissions structure if not found
		permissions = iam.ObjectPermissions{
			ObjectId:          objectId,
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

	objectType := req.Vars["object_type"]
	objectId := req.Vars["object_id"]

	if objectType == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "object_type is required"},
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

	permissionKey := fmt.Sprintf("%s:%s", objectType, objectId)

	// Get existing permissions or create new ones
	existingPermissions, exists := s.Permissions[permissionKey]
	if !exists {
		existingPermissions = iam.ObjectPermissions{
			ObjectId:          objectId,
			ObjectType:        objectType,
			AccessControlList: []iam.AccessControlResponse{},
		}
	}

	// Convert AccessControlRequest to AccessControlResponse
	var newAccessControlList []iam.AccessControlResponse
	for _, acl := range updateRequest.AccessControlList {
		response := iam.AccessControlResponse{
			UserName:             acl.UserName,
			GroupName:            acl.GroupName,
			ServicePrincipalName: acl.ServicePrincipalName,
			AllPermissions:       []iam.Permission{},
		}

		// Convert PermissionLevel to Permission
		if acl.PermissionLevel != "" {
			response.AllPermissions = append(response.AllPermissions, iam.Permission{
				PermissionLevel: acl.PermissionLevel,
			})
		}

		newAccessControlList = append(newAccessControlList, response)
	}

	// Update the permissions
	existingPermissions.AccessControlList = newAccessControlList
	s.Permissions[permissionKey] = existingPermissions

	return Response{
		Body: existingPermissions,
	}
}
