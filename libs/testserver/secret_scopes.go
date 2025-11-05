package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

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
	if _, exists := s.SecretScopes[request.Scope]; exists {
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

	s.SecretScopes[request.Scope] = scope

	return Response{}
}

func (s *FakeWorkspace) SecretsListScopes(req Request) Response {
	defer s.LockUnlock()()

	scopes := make([]workspace.SecretScope, 0, len(s.SecretScopes))
	for _, scope := range s.SecretScopes {
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

	if _, exists := s.SecretScopes[request.Scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", request.Scope)},
		}
	}

	delete(s.SecretScopes, request.Scope)
	// Also delete ACLs and secrets for this scope
	delete(s.Acls, request.Scope)
	delete(s.Secrets, request.Scope)

	return Response{}
}

func (s *FakeWorkspace) SecretsListAcls(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")

	// Check if scope exists
	if _, exists := s.SecretScopes[scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", scope)},
		}
	}

	acls := s.Acls[scope]
	if acls == nil {
		acls = []workspace.AclItem{}
	}

	return Response{
		Body: workspace.ListAclsResponse{
			Items: acls,
		},
	}
}
