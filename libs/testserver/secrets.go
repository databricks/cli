package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	secrets "github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"slices"
)

func (s *FakeWorkspace) SecretsPut(req Request) Response {
	defer s.LockUnlock()()

	// bytes_value arrives base64-encoded; reject malformed input the way the
	// API does instead of failing to decode the request.
	var raw struct {
		BytesValue string `json:"bytes_value"`
	}
	if err := json.Unmarshal(req.Body, &raw); err == nil && raw.BytesValue != "" {
		if _, err := base64.StdEncoding.DecodeString(raw.BytesValue); err != nil {
			return Response{
				StatusCode: 400,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "bytes_value must be base64-encoded"},
			}
		}
	}

	var request secrets.PutSecretRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 500,
		}
	}

	// Check if scope exists
	if _, exists := s.SecretScopes[val(request.Scope)]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Scope %s does not exist", val(request.Scope))},
		}
	}

	if s.Secrets == nil {
		s.Secrets = make(map[string]map[string]string)
	}
	if s.Secrets[val(request.Scope)] == nil {
		s.Secrets[val(request.Scope)] = make(map[string]string)
	}

	// Exactly one of string_value / bytes_value is set. bytes_value was decoded
	// from base64 with the request and GET re-encodes whatever was stored, so the
	// stored form is the raw secret in both cases.
	value := val(request.StringValue)
	if len(request.BytesValue) > 0 {
		value = string(request.BytesValue)
	}
	s.Secrets[val(request.Scope)][val(request.Key)] = value

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

	list := make([]secrets.SecretMetadata, 0, len(keys))
	for _, key := range keys {
		list = append(list, secrets.SecretMetadata{Key: nonZero(key)})
	}

	return Response{
		Body: secrets.ListSecretsResponse{
			Secrets: list,
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

	// encoding/json base64-encodes []byte, matching the server side behavior.
	return Response{
		Body: secrets.GetSecretResponse{
			Key:   nonZero(key),
			Value: []byte(secretValue),
		},
	}
}
