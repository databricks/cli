package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// maxSecretScopeNameLength is the backend's cap on a scope name.
const maxSecretScopeNameLength = 128

func (s *FakeWorkspace) SecretsCreateScope(req Request) Response {
	defer s.LockUnlock()()

	var request workspace.CreateScope
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// The name is validated before the collision check, so an empty one is refused on its own terms
	// rather than reported as colliding with a previous empty-named scope.
	if request.Scope == "" || len(request.Scope) > maxSecretScopeNameLength {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"message": fmt.Sprintf("Scope name must be non-empty and at most %d characters!", maxSecretScopeNameLength),
			},
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

	// An Azure Key Vault scope is backed by a real vault, so the API requires its metadata
	// and rejects the scope without it.
	if backendType == workspace.ScopeBackendTypeAzureKeyvault && request.BackendAzureKeyvault == nil {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "Scope with Azure KeyVault must have AzureKeyVaultSecretScopeMetadata defined!",
			},
		}
	}

	scope := workspace.SecretScope{
		Name:             request.Scope,
		BackendType:      backendType,
		KeyvaultMetadata: request.BackendAzureKeyvault,
	}

	s.SecretScopes[request.Scope] = scope

	// Automatically grant MANAGE permission to the creator (current user)
	// This matches real Databricks behavior
	if s.Acls == nil {
		s.Acls = make(map[string][]workspace.AclItem)
	}
	s.Acls[request.Scope] = []workspace.AclItem{
		{
			Principal:  s.CurrentUser().UserName,
			Permission: workspace.AclPermissionManage,
		},
	}

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
