package testserver

import (
	"encoding/json"
	"fmt"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/databricks/cli/libs/utils"
)

func (s *FakeWorkspace) SecretsAclsGet(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")
	principal := req.URL.Query().Get("principal")

	for _, acl := range s.Acls[scope] {
		if utils.Deref(acl.Principal) == principal {
			return Response{Body: acl}
		}
	}
	return Response{StatusCode: 404}
}

func (s *FakeWorkspace) SecretsAclsPut(req Request) Response {
	defer s.LockUnlock()()

	var request secrets.PutAclRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// If the ACL already exists, update it in place
	scopeAcls := s.Acls[utils.Deref(request.Scope)]
	for i, acl := range scopeAcls {
		if utils.Deref(acl.Principal) == utils.Deref(request.Principal) {
			s.Acls[utils.Deref(request.Scope)][i] = secrets.AclItem{
				Principal:  request.Principal,
				Permission: request.Permission,
			}
			return Response{}
		}
	}

	// Otherwise, add a new ACL
	s.Acls[utils.Deref(request.Scope)] = append(s.Acls[utils.Deref(request.Scope)], secrets.AclItem{
		Principal:  request.Principal,
		Permission: request.Permission,
	})
	return Response{}
}

func (s *FakeWorkspace) SecretsAclsDelete(req Request) Response {
	defer s.LockUnlock()()

	var request secrets.DeleteAclRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	scopeAcls := s.Acls[utils.Deref(request.Scope)]
	for i, acl := range scopeAcls {
		if utils.Deref(acl.Principal) == utils.Deref(request.Principal) {
			s.Acls[utils.Deref(request.Scope)] = append(scopeAcls[:i], scopeAcls[i+1:]...)
			return Response{}
		}
	}
	return Response{StatusCode: 404}
}
