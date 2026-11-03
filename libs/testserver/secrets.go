package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"

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

	// Exactly one of string_value / bytes_value is set. bytes_value arrives
	// already base64-encoded, and GET returns whatever was stored re-encoded,
	// so decode it here to keep the stored form the raw secret in both cases.
	value := request.StringValue
	if request.BytesValue != "" {
		decoded, err := base64.StdEncoding.DecodeString(request.BytesValue)
		if err != nil {
			return Response{
				StatusCode: 400,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "bytes_value must be base64-encoded"},
			}
		}
		value = string(decoded)
	}
	s.Secrets[request.Scope][request.Key] = value

	return Response{}
}

// SecretsList models GET /api/2.0/secrets/list. A missing scope must return
// RESOURCE_DOES_NOT_EXIST so callers can branch on apierr.ErrResourceDoesNotExist.
func (s *FakeWorkspace) SecretsList(req Request) Response {
	defer s.LockUnlock()()

	scope := req.URL.Query().Get("scope")

	if _, exists := s.SecretScopes[scope]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Scope %s does not exist", scope)},
		}
	}

	keys := make([]string, 0, len(s.Secrets[scope]))
	for key := range s.Secrets[scope] {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	secrets := make([]workspace.SecretMetadata, 0, len(keys))
	for _, key := range keys {
		secrets = append(secrets, workspace.SecretMetadata{Key: key})
	}

	return Response{
		Body: workspace.ListSecretsResponse{
			Secrets: secrets,
		},
	}
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
