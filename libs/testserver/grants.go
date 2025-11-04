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
	current := make(map[string]map[string]struct{})
	for _, assignment := range s.Grants[key] {
		privs := current[assignment.Principal]
		if privs == nil {
			privs = make(map[string]struct{})
			current[assignment.Principal] = privs
		}
		for _, privilege := range assignment.Privileges {
			privs[string(privilege)] = struct{}{}
		}
	}

	for _, change := range request.Changes {
		if change.Principal == "" {
			continue
		}
		privs := current[change.Principal]
		if privs == nil {
			privs = make(map[string]struct{})
			current[change.Principal] = privs
		}
		for _, privilege := range change.Remove {
			delete(privs, string(privilege))
		}
		for _, privilege := range change.Add {
			privs[string(privilege)] = struct{}{}
		}
	}

	var assignments []catalog.PrivilegeAssignment
	for principal, privs := range current {
		if len(privs) == 0 {
			continue
		}
		privileges := make([]catalog.Privilege, 0, len(privs))
		for privilege := range privs {
			privileges = append(privileges, catalog.Privilege(privilege))
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
