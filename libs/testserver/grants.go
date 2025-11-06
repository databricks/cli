package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func grantsKey(securableType, fullName string) string {
	return strings.ToUpper(securableType) + ":" + fullName
}

func (s *FakeWorkspace) GrantsUpdate(req Request, securableType, fullName string) Response {
	var request catalog.UpdatePermissions
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	key := grantsKey(securableType, fullName)

	defer s.LockUnlock()()

	// Build a simple map of principals to privileges
	principalPrivs := make(map[string]map[string]bool)

	// Load current grants
	for _, assignment := range s.Grants[key] {
		if principalPrivs[assignment.Principal] == nil {
			principalPrivs[assignment.Principal] = make(map[string]bool)
		}
		for _, privilege := range assignment.Privileges {
			principalPrivs[assignment.Principal][string(privilege)] = true
		}
	}

	// Apply changes
	for _, change := range request.Changes {
		if change.Principal == "" {
			continue
		}
		if principalPrivs[change.Principal] == nil {
			principalPrivs[change.Principal] = make(map[string]bool)
		}

		// Remove privileges
		for _, privilege := range change.Remove {
			if privilege == catalog.PrivilegeAllPrivileges {
				principalPrivs[change.Principal] = make(map[string]bool)
			} else {
				delete(principalPrivs[change.Principal], string(privilege))
			}
		}

		// Add privileges
		for _, privilege := range change.Add {
			principalPrivs[change.Principal][string(privilege)] = true
		}
	}

	// Convert back to assignments with sorted privileges
	var assignments []catalog.PrivilegeAssignment
	for principal, privs := range principalPrivs {
		if len(privs) == 0 {
			continue
		}

		// Sort privileges alphabetically
		var privilegeStrs []string
		for priv := range privs {
			privilegeStrs = append(privilegeStrs, priv)
		}
		slices.Sort(privilegeStrs)

		privileges := make([]catalog.Privilege, len(privilegeStrs))
		for i, priv := range privilegeStrs {
			privileges[i] = catalog.Privilege(priv)
		}

		assignments = append(assignments, catalog.PrivilegeAssignment{
			Principal:  principal,
			Privileges: privileges,
		})
	}

	s.Grants[key] = assignments

	return Response{Body: catalog.UpdatePermissionsResponse{PrivilegeAssignments: assignments}}
}

func (s *FakeWorkspace) GrantsGet(_ Request, securableType, fullName string) Response {
	defer s.LockUnlock()()

	key := grantsKey(securableType, fullName)
	assignments := s.Grants[key]

	return Response{Body: catalog.GetPermissionsResponse{PrivilegeAssignments: assignments}}
}
