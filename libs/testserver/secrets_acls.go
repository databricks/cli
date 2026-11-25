package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerRoutes("secrets_acls", addSecretsAclsRoutes)
	registerStore("secrets_acls", newSecretsAclsStore)
}

type secretsAclsStore struct {
	Acls map[string][]workspace.AclItem
}

func newSecretsAclsStore(*FakeWorkspace) *secretsAclsStore {
	return &secretsAclsStore{
		Acls: map[string][]workspace.AclItem{},
	}
}

func (s *FakeWorkspace) secretsAcls() *secretsAclsStore {
	return stateOf[secretsAclsStore](s)
}

func addSecretsAclsRoutes(server *Server) {
	// Secrets ACLs:
	server.Handle("GET", "/api/2.0/secrets/acls/get", func(req Request) any {
		return req.Workspace.SecretsAclsGet(req)
	})

	server.Handle("GET", "/api/2.0/secrets/acls/list", func(req Request) any {
		return req.Workspace.SecretsListAcls(req)
	})

	server.Handle("POST", "/api/2.0/secrets/acls/put", func(req Request) any {
		return req.Workspace.SecretsAclsPut(req)
	})

	server.Handle("POST", "/api/2.0/secrets/acls/delete", func(req Request) any {
		return req.Workspace.SecretsAclsDelete(req)
	})
}

func (s *FakeWorkspace) SecretsAclsGet(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")
	principal := req.URL.Query().Get("principal")

	for _, acl := range s.secretsAcls().Acls[scope] {
		if acl.Principal == principal {
			return Response{Body: acl}
		}
	}
	return Response{StatusCode: 404}
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

	// If the ACL already exists, update it in place
	scopeAcls := s.secretsAcls().Acls[request.Scope]
	for i, acl := range scopeAcls {
		if acl.Principal == request.Principal {
			s.secretsAcls().Acls[request.Scope][i] = workspace.AclItem{
				Principal:  request.Principal,
				Permission: request.Permission,
			}
			return Response{}
		}
	}

	// Otherwise, add a new ACL
	s.secretsAcls().Acls[request.Scope] = append(s.secretsAcls().Acls[request.Scope], workspace.AclItem{
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

	scopeAcls := s.secretsAcls().Acls[request.Scope]
	for i, acl := range scopeAcls {
		if acl.Principal == request.Principal {
			s.secretsAcls().Acls[request.Scope] = append(scopeAcls[:i], scopeAcls[i+1:]...)
			return Response{}
		}
	}
	return Response{StatusCode: 404}
}
