package testserver

import (
	"encoding/json"
	"fmt"
	grants "github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"net/http"
	"slices"
	"strings"
)

// notImpliedByAllPrivileges are the privileges UC does not include in
// ALL_PRIVILEGES, so granting or revoking ALL_PRIVILEGES never affects them.
var notImpliedByAllPrivileges = map[string]bool{
	"MANAGE":                true,
	"READ_METADATA":         true,
	"EXTERNAL_USE_SCHEMA":   true,
	"EXTERNAL_USE_LOCATION": true,
}

func grantsKey(securableType, fullName string) string {
	return strings.ToUpper(securableType) + ":" + fullName
}

func (s *FakeWorkspace) GrantsUpdate(req Request, securableType, fullName string) Response {
	var request grants.UpdatePermissionsRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	key := grantsKey(securableType, fullName)

	defer s.LockUnlock()()

	// Build a map of principals to privileges (using Privilege type directly)
	principalPrivs := make(map[string]map[string]bool)

	// Load current grants
	for _, assignment := range s.Grants[key] {
		if principalPrivs[val(assignment.Principal)] == nil {
			principalPrivs[val(assignment.Principal)] = make(map[string]bool)
		}
		for _, privilege := range assignment.Privileges {
			principalPrivs[val(assignment.Principal)][privilege] = true
		}
	}

	// Validate: reject duplicate privileges in Add and Remove for the same principal
	for _, change := range request.Changes {
		addSet := make(map[string]bool, len(change.Add))
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
		if val(change.Principal) == "" {
			continue
		}
		if principalPrivs[val(change.Principal)] == nil {
			principalPrivs[val(change.Principal)] = make(map[string]bool)
		}

		// Remove privileges
		for _, privilege := range change.Remove {
			if privilege == "ALL_PRIVILEGES" {
				// Removing ALL_PRIVILEGES clears every privilege it implies, but NOT
				// the ones UC does not include in ALL_PRIVILEGES (MANAGE,
				// READ_METADATA, EXTERNAL_USE_SCHEMA, EXTERNAL_USE_LOCATION). Those
				// are granted independently and survive. See
				// https://docs.databricks.com/aws/en/data-governance/unity-catalog/manage-privileges/privileges
				for p := range principalPrivs[val(change.Principal)] {
					if !notImpliedByAllPrivileges[p] {
						delete(principalPrivs[val(change.Principal)], p)
					}
				}
			} else {
				delete(principalPrivs[val(change.Principal)], privilege)
			}
		}

		// Add privileges
		for _, privilege := range change.Add {
			principalPrivs[val(change.Principal)][privilege] = true
		}
	}

	// Convert back to assignments with sorted privileges
	// Note order of assignments is randomized due to map. This is intentional, azure backend behaves the same way
	// (deco env run -i -n azure-prod-ucws -- go test ./acceptance -run ^TestAccept$/^bundle$/^resources$/^grants$/^schemas$/^out_of_band_principal$/direct -count=10 -failfast -timeout=1h)

	var assignments []grants.PrivilegeAssignment
	for principal, privs := range principalPrivs {
		if len(privs) == 0 {
			continue
		}

		// Collect and sort privileges directly
		privileges := make([]string, 0, len(privs))
		for priv := range privs {
			privileges = append(privileges, priv)
		}
		slices.SortFunc(privileges, func(a, b string) int {
			return strings.Compare(string(a), string(b))
		})

		assignments = append(assignments, grants.PrivilegeAssignment{
			Principal:  nonZero(principal),
			Privileges: privileges,
		})
	}

	s.Grants[key] = assignments

	if securableType == "schema" {
		schema, ok := s.Schemas[fullName]
		if ok {
			schema.UpdatedAt = nonZero(nowMilli())
			schema.UpdatedBy = s.CurrentUser().UserName
			s.Schemas[fullName] = schema
		}
	}

	return Response{Body: grants.UpdatePermissionsResponse{PrivilegeAssignments: assignments}}
}

func (s *FakeWorkspace) GrantsGet(_ Request, securableType, fullName string) Response {
	defer s.LockUnlock()()

	key := grantsKey(securableType, fullName)
	assignments := s.Grants[key]

	return Response{Body: grants.GetPermissionsResponse{PrivilegeAssignments: assignments}}
}
