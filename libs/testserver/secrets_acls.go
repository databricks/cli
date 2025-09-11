package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func (s *FakeWorkspace) SecretsAclsGet(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")
	principal := req.URL.Query().Get("principal")

	for _, acl := range s.Acls[scope] {
		if acl.Principal == principal {
			return Response{Body: acl}
		}
	}
	return Response{StatusCode: 404}
}

func (s *FakeWorkspace) SecretsAclsList(req Request) Response {
	scope := req.URL.Query().Get("scope")
	return MapGet(s, s.Acls, scope)
}

func (s *FakeWorkspace) SecretsAclsPut(req Request) Response {
	defer s.LockUnlock()()

	var request workspace.PutAcl
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	s.Acls[request.Scope] = append(s.Acls[request.Scope], workspace.AclItem{
		Principal:  request.Principal,
		Permission: request.Permission,
	})
	return Response{}
}

func (s *FakeWorkspace) SecretsAclsDelete(req Request) Response {
	defer s.LockUnlock()()

	var request workspace.DeleteAcl
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	scopeAcls := s.Acls[request.Scope]
	for i, acl := range scopeAcls {
		if acl.Principal == request.Principal {
			s.Acls[request.Scope] = append(scopeAcls[:i], scopeAcls[i+1:]...)
			return Response{}
		}
	}
	return Response{StatusCode: 404}
}
