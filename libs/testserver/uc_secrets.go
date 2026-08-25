package testserver

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// SecretsUcCreateSecret handles POST /api/2.1/unity-catalog/secrets
func (s *FakeWorkspace) SecretsUcCreateSecret(req Request) Response {
	defer s.LockUnlock()()

	// The API accepts flat fields, not wrapped in a "secret" object
	var inputSecret catalog.Secret
	if err := json.Unmarshal(req.Body, &inputSecret); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	if s.UCSecrets == nil {
		s.UCSecrets = make(map[string]catalog.Secret)
	}

	fullName := fmt.Sprintf("%s.%s.%s",
		inputSecret.CatalogName,
		inputSecret.SchemaName,
		inputSecret.Name)

	// Check if secret already exists
	if _, exists := s.UCSecrets[fullName]; exists {
		return Response{
			StatusCode: 409,
			Body: map[string]any{
				"error_code": "RESOURCE_ALREADY_EXISTS",
				"message":    fmt.Sprintf("Secret %s already exists", fullName),
			},
		}
	}

	now := sdktime.New(time.Now())
	secret := catalog.Secret{
		CatalogName:    inputSecret.CatalogName,
		SchemaName:     inputSecret.SchemaName,
		Name:           inputSecret.Name,
		FullName:       fullName,
		Value:          inputSecret.Value,
		EffectiveValue: inputSecret.Value,
		Comment:        inputSecret.Comment,
		Owner:          inputSecret.Owner,
		ExpireTime:     inputSecret.ExpireTime,
		CreateTime:     now,
		UpdateTime:     now,
		CreatedBy:      "test-user@databricks.com",
		UpdatedBy:      "test-user@databricks.com",
		EffectiveOwner: inputSecret.Owner,
		MetastoreId:    "test-metastore-id",
	}

	if secret.Owner == "" {
		secret.Owner = "test-user@databricks.com"
		secret.EffectiveOwner = "test-user@databricks.com"
	}

	s.UCSecrets[fullName] = secret

	return Response{
		Body: secret,
	}
}

// SecretsUcGetSecret handles GET /api/2.1/unity-catalog/secrets/{full_name}
func (s *FakeWorkspace) SecretsUcGetSecret(req Request) Response {
	defer s.LockUnlock()()

	// Extract full_name from path parameter
	fullName := req.Vars["full_name"]
	if fullName == "" {
		// Fallback: extract from path
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) >= 6 {
			fullName = parts[5]
		}
	}

	secret, exists := s.UCSecrets[fullName]
	if !exists {
		return Response{
			StatusCode: 404,
			Body: map[string]any{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Secret %s not found", fullName),
			},
		}
	}

	// Return secret without the actual value (only metadata)
	// The real API doesn't return the value unless specifically requested
	returnSecret := secret
	returnSecret.EffectiveValue = secret.Value
	returnSecret.Value = ""

	return Response{
		Body: returnSecret,
	}
}

// SecretsUcUpdateSecret handles PATCH /api/2.1/unity-catalog/secrets/{full_name}
func (s *FakeWorkspace) SecretsUcUpdateSecret(req Request) Response {
	defer s.LockUnlock()()

	// Extract full_name from path parameter
	fullName := req.Vars["full_name"]
	if fullName == "" {
		// Fallback: extract from path
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) >= 6 {
			fullName = parts[5]
		}
	}

	// The API accepts flat fields
	var updateSecret catalog.Secret
	if err := json.Unmarshal(req.Body, &updateSecret); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	secret, exists := s.UCSecrets[fullName]
	if !exists {
		return Response{
			StatusCode: 404,
			Body: map[string]any{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Secret %s not found", fullName),
			},
		}
	}

	// Update fields based on update mask
	if updateSecret.Value != "" {
		secret.Value = updateSecret.Value
	}
	if updateSecret.Comment != "" {
		secret.Comment = updateSecret.Comment
	}
	if updateSecret.Owner != "" {
		secret.Owner = updateSecret.Owner
		secret.EffectiveOwner = updateSecret.Owner
	}
	// The CLI masks the whole secret with update_mask=*, so an absent expire_time
	// clears the existing one rather than leaving it in place.
	secret.ExpireTime = updateSecret.ExpireTime

	secret.UpdateTime = sdktime.New(time.Now())
	secret.UpdatedBy = "test-user@databricks.com"

	s.UCSecrets[fullName] = secret

	return Response{
		Body: secret,
	}
}

// SecretsUcDeleteSecret handles DELETE /api/2.1/unity-catalog/secrets/{full_name}
func (s *FakeWorkspace) SecretsUcDeleteSecret(req Request) Response {
	defer s.LockUnlock()()

	// Extract full_name from path parameter
	fullName := req.Vars["full_name"]
	if fullName == "" {
		// Fallback: extract from path
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) >= 6 {
			fullName = parts[5]
		}
	}

	if _, exists := s.UCSecrets[fullName]; !exists {
		return Response{
			StatusCode: 404,
			Body: map[string]any{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    fmt.Sprintf("Secret %s not found", fullName),
			},
		}
	}

	delete(s.UCSecrets, fullName)

	return Response{}
}
