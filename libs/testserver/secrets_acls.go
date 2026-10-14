package testserver

import (
	"encoding/json"
	"fmt"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
)

func (s *FakeWorkspace) SecretsAclsGet(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")
	principal := req.URL.Query().Get("principal")

	for _, acl := range s.Acls[scope] {
		if val(acl.Principal) == principal {
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
	scopeAcls := s.Acls[val(request.Scope)]
	for i, acl := range scopeAcls {
		if acl.Principal == request.Principal {
			s.Acls[val(request.Scope)][i] = secrets.AclItem{
				Principal:  request.Principal,
				Permission: request.Permission,
			}
			return Response{}
		}
	}

	// Otherwise, add a new ACL
	s.Acls[val(request.Scope)] = append(s.Acls[val(request.Scope)], secrets.AclItem{
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

	scopeAcls := s.Acls[val(request.Scope)]
	for i, acl := range scopeAcls {
		if acl.Principal == request.Principal {
			s.Acls[val(request.Scope)] = append(scopeAcls[:i], scopeAcls[i+1:]...)
			return Response{}
		}
	}
	return Response{StatusCode: 404}
}
