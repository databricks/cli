package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// SecretScopeCreate handles POST /api/2.0/secrets/scopes/create
func (s *FakeWorkspace) SecretScopeCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq workspace.CreateScope
	if err := json.Unmarshal(req.Body, &createReq); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": err.Error()},
		}
	}

	if createReq.Scope == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "Scope name is required"},
		}
	}

	if _, exists := s.SecretScopes[createReq.Scope]; exists {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": "Scope already exists"},
		}
	}

	scope := workspace.SecretScope{
		Name:             createReq.Scope,
		BackendType:      createReq.ScopeBackendType,
		KeyvaultMetadata: createReq.BackendAzureKeyvault,
	}

	s.SecretScopes[createReq.Scope] = scope

	return Response{}
}

// SecretScopeList handles GET /api/2.0/secrets/scopes/list
func (s *FakeWorkspace) SecretScopeList(req Request) Response {
	defer s.LockUnlock()()

	scopes := make([]workspace.SecretScope, 0, len(s.SecretScopes))
	for _, scope := range s.SecretScopes {
		scopes = append(scopes, scope)
	}

	return Response{
		Body: map[string]any{
			"scopes": scopes,
		},
	}
}

// SecretScopeDelete handles POST /api/2.0/secrets/scopes/delete
func (s *FakeWorkspace) SecretScopeDelete(req Request) Response {
	defer s.LockUnlock()()

	var deleteReq struct {
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(req.Body, &deleteReq); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": err.Error()},
		}
	}

	if _, exists := s.SecretScopes[deleteReq.Scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Scope not found"},
		}
	}

	delete(s.SecretScopes, deleteReq.Scope)

	// Also delete all secrets in this scope
	for key := range s.Secrets {
		// Key format is "scope/key"
		if len(key) > len(deleteReq.Scope) && key[:len(deleteReq.Scope)] == deleteReq.Scope && key[len(deleteReq.Scope)] == '/' {
			delete(s.Secrets, key)
		}
	}

	return Response{}
}

// SecretPut handles POST /api/2.0/secrets/put
func (s *FakeWorkspace) SecretPut(req Request) Response {
	defer s.LockUnlock()()

	var putReq workspace.PutSecret
	if err := json.Unmarshal(req.Body, &putReq); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": err.Error()},
		}
	}

	if putReq.Scope == "" || putReq.Key == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "Scope and key are required"},
		}
	}

	// Check if scope exists
	if _, exists := s.SecretScopes[putReq.Scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Scope not found"},
		}
	}

	secretKey := fmt.Sprintf("%s/%s", putReq.Scope, putReq.Key)
	s.Secrets[secretKey] = workspace.SecretMetadata{
		Key:                  putReq.Key,
		LastUpdatedTimestamp: nowMilli(),
	}

	return Response{}
}

// SecretGet handles GET /api/2.0/secrets/get
func (s *FakeWorkspace) SecretGet(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")
	key := req.URL.Query().Get("key")

	if scope == "" || key == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": "Scope and key are required"},
		}
	}

	secretKey := fmt.Sprintf("%s/%s", scope, key)
	secret, exists := s.Secrets[secretKey]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Secret not found"},
		}
	}

	return Response{
		Body: workspace.GetSecretResponse{
			Key:   secret.Key,
			Value: "", // Value is never returned for security reasons
		},
	}
}

// SecretDelete handles POST /api/2.0/secrets/delete
func (s *FakeWorkspace) SecretDelete(req Request) Response {
	defer s.LockUnlock()()

	var deleteReq workspace.DeleteSecret
	if err := json.Unmarshal(req.Body, &deleteReq); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": err.Error()},
		}
	}

	secretKey := fmt.Sprintf("%s/%s", deleteReq.Scope, deleteReq.Key)
	if _, exists := s.Secrets[secretKey]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Secret not found"},
		}
	}

	delete(s.Secrets, secretKey)

	return Response{}
}
