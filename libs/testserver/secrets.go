package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func (s *FakeWorkspace) SecretsPut(req Request) Response {
	defer s.LockUnlock()()

	var request workspace.PutSecret
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// Check if scope exists
	if _, exists := s.SecretScopes[request.Scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", request.Scope)},
		}
	}

	if s.Secrets == nil {
		s.Secrets = make(map[string]map[string]string)
	}
	if s.Secrets[request.Scope] == nil {
		s.Secrets[request.Scope] = make(map[string]string)
	}

	// Store the secret value
	s.Secrets[request.Scope][request.Key] = request.StringValue

	return Response{}
}

func (s *FakeWorkspace) SecretsGet(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")
	key := req.URL.Query().Get("key")

	// Check if scope exists
	if _, exists := s.SecretScopes[scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", scope)},
		}
	}

	// Check if secret exists
	if s.Secrets == nil || s.Secrets[scope] == nil {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Secret %s/%s not found", scope, key)},
		}
	}

	secretValue, exists := s.Secrets[scope][key]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Secret %s/%s not found", scope, key)},
		}
	}

	// Base64 encode the secret value, to match the server side behavior.
	encodedValue := base64.StdEncoding.EncodeToString([]byte(secretValue))

	return Response{
		Body: workspace.GetSecretResponse{
			Key:   key,
			Value: encodedValue,
		},
	}
}
