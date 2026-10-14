package testserver

import (
	"encoding/json"
	"fmt"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/databricks/cli/libs/utils"
)

func (s *FakeWorkspace) SecretsCreateScope(req Request) Response {
	defer s.LockUnlock()()

	var request secrets.CreateScopeRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// Check if scope already exists
	if _, exists := s.SecretScopes[utils.Deref(request.Scope)]; exists {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s already exists", utils.Deref(request.Scope))},
		}
	}

	// Create the scope
	backendType := request.ScopeBackendType
	if backendType == "" {
		backendType = secrets.ScopeBackendType_Databricks
	}

	scope := secrets.SecretScope{
		Name:             request.Scope,
		BackendType:      backendType,
		KeyvaultMetadata: request.BackendAzureKeyvault,
	}

	s.SecretScopes[utils.Deref(request.Scope)] = scope

	// Automatically grant MANAGE permission to the creator (current user)
	// This matches real Databricks behavior
	if s.Acls == nil {
		s.Acls = make(map[string][]secrets.AclItem)
	}
	s.Acls[utils.Deref(request.Scope)] = []secrets.AclItem{
		{
			Principal:  s.CurrentUser().UserName,
			Permission: secrets.AclPermission_Manage,
		},
	}

	return Response{}
}

func (s *FakeWorkspace) SecretsListScopes(req Request) Response {
	defer s.LockUnlock()()

	scopes := make([]secrets.SecretScope, 0, len(s.SecretScopes))
	for _, scope := range s.SecretScopes {
		scopes = append(scopes, scope)
	}

	return Response{
		Body: secrets.ListScopesResponse{
			Scopes: scopes,
		},
	}
}

func (s *FakeWorkspace) SecretsDeleteScope(req Request) Response {
	defer s.LockUnlock()()

	var request secrets.DeleteScopeRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	if _, exists := s.SecretScopes[utils.Deref(request.Scope)]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", utils.Deref(request.Scope))},
		}
	}

	delete(s.SecretScopes, utils.Deref(request.Scope))
	// Also delete ACLs and secrets for this scope
	delete(s.Acls, utils.Deref(request.Scope))
	delete(s.Secrets, utils.Deref(request.Scope))

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
		acls = []secrets.AclItem{}
	}

	return Response{
		Body: secrets.ListAclsResponse{
			Items: acls,
		},
	}
}
