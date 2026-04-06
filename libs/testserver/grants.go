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

	// Build a map of principals to privileges (using Privilege type directly)
	principalPrivs := make(map[string]map[catalog.Privilege]bool)

	// Load current grants
	for _, assignment := range s.Grants[key] {
		if principalPrivs[assignment.Principal] == nil {
			principalPrivs[assignment.Principal] = make(map[catalog.Privilege]bool)
		}
		for _, privilege := range assignment.Privileges {
			principalPrivs[assignment.Principal][privilege] = true
		}
	}

	// Validate: reject duplicate privileges in Add and Remove for the same principal
	for _, change := range request.Changes {
		addSet := make(map[catalog.Privilege]bool, len(change.Add))
		for _, p := range change.Add {
			addSet[p] = true
		}
		for _, p := range change.Remove {
			if addSet[p] {
				return Response{
					StatusCode: http.StatusBadRequest,
					Body: map[string]string{
						"error_code": "INVALID_PARAMETER_VALUE",
						"message":    fmt.Sprintf("Duplicate privileges to add and delete for principal %s.", change.Principal),
					},
				}
			}
		}
	}

	// Apply changes
	for _, change := range request.Changes {
		if change.Principal == "" {
			continue
		}
		if principalPrivs[change.Principal] == nil {
			principalPrivs[change.Principal] = make(map[catalog.Privilege]bool)
		}

		// Remove privileges
		for _, privilege := range change.Remove {
			if privilege == catalog.PrivilegeAllPrivileges {
				principalPrivs[change.Principal] = make(map[catalog.Privilege]bool)
			} else {
				delete(principalPrivs[change.Principal], privilege)
			}
		}

		// Add privileges
		for _, privilege := range change.Add {
			principalPrivs[change.Principal][privilege] = true
		}
	}

	// Convert back to assignments with sorted privileges
	// Note order of assignments is randomized due to map. This is intentional, azure backend behaves the same way
	// (deco env run -i -n azure-prod-ucws -- go test ./acceptance -run ^TestAccept$/^bundle$/^resources$/^grants$/^schemas$/^out_of_band_principal$/direct -count=10 -failfast -timeout=1h)

	var assignments []catalog.PrivilegeAssignment
	for principal, privs := range principalPrivs {
		if len(privs) == 0 {
			continue
		}

		// Collect and sort privileges directly
		privileges := make([]catalog.Privilege, 0, len(privs))
		for priv := range privs {
			privileges = append(privileges, priv)
		}
		slices.SortFunc(privileges, func(a, b catalog.Privilege) int {
			return strings.Compare(string(a), string(b))
		})

		assignments = append(assignments, catalog.PrivilegeAssignment{
			Principal:  principal,
			Privileges: privileges,
		})
	}

	s.Grants[key] = assignments

	if securableType == "schema" {
		schema, ok := s.Schemas[fullName]
		if ok {
			schema.UpdatedAt = nowMilli()
			schema.UpdatedBy = s.CurrentUser().UserName
			s.Schemas[fullName] = schema
		}
	}

	return Response{Body: catalog.UpdatePermissionsResponse{PrivilegeAssignments: assignments}}
}

func (s *FakeWorkspace) GrantsGet(_ Request, securableType, fullName string) Response {
	defer s.LockUnlock()()

	key := grantsKey(securableType, fullName)
	assignments := s.Grants[key]

	return Response{Body: catalog.GetPermissionsResponse{PrivilegeAssignments: assignments}}
}
