package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	// Use ordered structures to preserve insertion order
	type principalPrivs struct {
		principal  string
		privileges []string
	}
	var orderedCurrent []principalPrivs
	principalIndex := make(map[string]int)

	for _, assignment := range s.Grants[key] {
		if idx, exists := principalIndex[assignment.Principal]; exists {
			// Principal exists, append privileges
			for _, privilege := range assignment.Privileges {
				orderedCurrent[idx].privileges = append(orderedCurrent[idx].privileges, string(privilege))
			}
		} else {
			// New principal
			principalIndex[assignment.Principal] = len(orderedCurrent)
			privileges := make([]string, len(assignment.Privileges))
			for i, privilege := range assignment.Privileges {
				privileges[i] = string(privilege)
			}
			orderedCurrent = append(orderedCurrent, principalPrivs{
				principal:  assignment.Principal,
				privileges: privileges,
			})
		}
	}

	for _, change := range request.Changes {
		if change.Principal == "" {
			continue
		}
		if idx, exists := principalIndex[change.Principal]; exists {
			// Principal exists, modify their privileges
			privs := &orderedCurrent[idx].privileges

			// Remove privileges
			for _, privilege := range change.Remove {
				privilegeStr := string(privilege)
				for i := 0; i < len(*privs); i++ {
					if (*privs)[i] == privilegeStr {
						*privs = append((*privs)[:i], (*privs)[i+1:]...)
						i-- // Adjust index after removal
					}
				}
			}

			// Add privileges
			for _, privilege := range change.Add {
				*privs = append(*privs, string(privilege))
			}
		} else {
			// New principal
			principalIndex[change.Principal] = len(orderedCurrent)
			privileges := make([]string, len(change.Add))
			for i, privilege := range change.Add {
				privileges[i] = string(privilege)
			}
			orderedCurrent = append(orderedCurrent, principalPrivs{
				principal:  change.Principal,
				privileges: privileges,
			})
		}
	}

	var assignments []catalog.PrivilegeAssignment
	for _, entry := range orderedCurrent {
		if len(entry.privileges) == 0 {
			continue
		}
		privileges := make([]catalog.Privilege, len(entry.privileges))
		for i, privilege := range entry.privileges {
			privileges[i] = catalog.Privilege(privilege)
		}
		assignments = append(assignments, catalog.PrivilegeAssignment{
			Principal:  entry.principal,
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
