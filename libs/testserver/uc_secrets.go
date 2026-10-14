package testserver

import (
	"encoding/json"
	"fmt"
	ucsecrets "github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	"github.com/databricks/cli/libs/utils"
	"strings"
	"time"

	"github.com/databricks/sdk-go/core/types"
)

// SecretsUcCreateSecret handles POST /api/2.1/unity-catalog/secrets
func (s *FakeWorkspace) SecretsUcCreateSecret(req Request) Response {
	defer s.LockUnlock()()

	// The API accepts flat fields, not wrapped in a "secret" object
	var inputSecret ucsecrets.Secret
	if err := json.Unmarshal(req.Body, &inputSecret); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	if s.UCSecrets == nil {
		s.UCSecrets = make(map[string]ucsecrets.Secret)
	}

	fullName := fmt.Sprintf("%s.%s.%s",
		utils.Deref(inputSecret.CatalogName),
		utils.Deref(inputSecret.SchemaName),
		utils.Deref(inputSecret.Name))

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

	now := types.NewFromTime(time.Now())
	secret := ucsecrets.Secret{
		CatalogName:    inputSecret.CatalogName,
		SchemaName:     inputSecret.SchemaName,
		Name:           inputSecret.Name,
		FullName:       utils.NonZero(fullName),
		Value:          inputSecret.Value,
		EffectiveValue: inputSecret.Value,
		Comment:        inputSecret.Comment,
		Owner:          inputSecret.Owner,
		ExpireTime:     inputSecret.ExpireTime,
		CreateTime:     now,
		UpdateTime:     now,
		CreatedBy:      s.CurrentUser().UserName,
		UpdatedBy:      s.CurrentUser().UserName,
		EffectiveOwner: inputSecret.Owner,
		MetastoreId:    utils.NonZero("test-metastore-id"),
	}

	if utils.Deref(secret.Owner) == "" {
		secret.Owner = s.CurrentUser().UserName
		secret.EffectiveOwner = s.CurrentUser().UserName
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
	returnSecret.Value = nil

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
	var updateSecret ucsecrets.Secret
	if err := json.Unmarshal(req.Body, &updateSecret); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	// Which fields the caller actually sent, so an explicit "comment": "" (the CLI force-sends it
	// under update_mask=*) clears the value instead of being treated as "not provided".
	fields, _ := parseUpdateFields(req.Body)

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
	if utils.Deref(updateSecret.Value) != "" {
		secret.Value = updateSecret.Value
	}
	if _, ok := fields["comment"]; ok {
		secret.Comment = updateSecret.Comment
	}
	if utils.Deref(updateSecret.Owner) != "" {
		secret.Owner = updateSecret.Owner
		secret.EffectiveOwner = updateSecret.Owner
	}
	// Verified against a real workspace: even with update_mask=*, an absent
	// expire_time leaves the existing one in place rather than clearing it, so a
	// removal never takes effect and the next plan asks for it again.
	if updateSecret.ExpireTime != nil {
		secret.ExpireTime = updateSecret.ExpireTime
	}

	secret.UpdateTime = types.NewFromTime(time.Now())
	secret.UpdatedBy = s.CurrentUser().UserName

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
