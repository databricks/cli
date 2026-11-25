package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func init() {
	registerRoutes("secret_scopes", addSecretScopesRoutes)
	registerStore("secret_scopes", newSecretScopesStore)
}

type secretScopesStore struct {
	SecretScopes map[string]workspace.SecretScope
}

func newSecretScopesStore(*FakeWorkspace) *secretScopesStore {
	return &secretScopesStore{
		SecretScopes: map[string]workspace.SecretScope{},
	}
}

func (s *FakeWorkspace) secretScopes() *secretScopesStore {
	return stateOf[secretScopesStore](s)
}

func addSecretScopesRoutes(server *Server) {
	// Secret Scopes:
	server.Handle("POST", "/api/2.0/secrets/scopes/create", func(req Request) any {
		return req.Workspace.SecretsCreateScope(req)
	})

	server.Handle("GET", "/api/2.0/secrets/scopes/list", func(req Request) any {
		return req.Workspace.SecretsListScopes(req)
	})

	server.Handle("POST", "/api/2.0/secrets/scopes/delete", func(req Request) any {
		return req.Workspace.SecretsDeleteScope(req)
	})
}

func (s *FakeWorkspace) SecretsCreateScope(req Request) Response {
	defer s.LockUnlock()()

	var request workspace.CreateScope
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// Check if scope already exists
	if _, exists := s.secretScopes().SecretScopes[request.Scope]; exists {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s already exists", request.Scope)},
		}
	}

	// Create the scope
	backendType := request.ScopeBackendType
	if backendType == "" {
		backendType = workspace.ScopeBackendTypeDatabricks
	}

	scope := workspace.SecretScope{
		Name:             request.Scope,
		BackendType:      backendType,
		KeyvaultMetadata: request.BackendAzureKeyvault,
	}

	s.secretScopes().SecretScopes[request.Scope] = scope

	// Automatically grant MANAGE permission to the creator (current user)
	// This matches real Databricks behavior
	if s.secretsAcls().Acls == nil {
		s.secretsAcls().Acls = make(map[string][]workspace.AclItem)
	}
	s.secretsAcls().Acls[request.Scope] = []workspace.AclItem{
		{
			Principal:  s.CurrentUser().UserName,
			Permission: workspace.AclPermissionManage,
		},
	}

	return Response{}
}

func (s *FakeWorkspace) SecretsListScopes(req Request) Response {
	defer s.LockUnlock()()

	scopes := make([]workspace.SecretScope, 0, len(s.secretScopes().SecretScopes))
	for _, scope := range s.secretScopes().SecretScopes {
		scopes = append(scopes, scope)
	}

	return Response{
		Body: workspace.ListScopesResponse{
			Scopes: scopes,
		},
	}
}

func (s *FakeWorkspace) SecretsDeleteScope(req Request) Response {
	defer s.LockUnlock()()

	var request workspace.DeleteScope
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	if _, exists := s.secretScopes().SecretScopes[request.Scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", request.Scope)},
		}
	}

	delete(s.secretScopes().SecretScopes, request.Scope)
	// Also delete ACLs and secrets for this scope
	delete(s.secretsAcls().Acls, request.Scope)
	delete(s.secrets().Secrets, request.Scope)

	return Response{}
}

func (s *FakeWorkspace) SecretsListAcls(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")

	// Check if scope exists
	if _, exists := s.secretScopes().SecretScopes[scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", scope)},
		}
	}

	acls := s.secretsAcls().Acls[scope]
	if acls == nil {
		acls = []workspace.AclItem{}
	}

	return Response{
		Body: workspace.ListAclsResponse{
			Items: acls,
		},
	}
}
