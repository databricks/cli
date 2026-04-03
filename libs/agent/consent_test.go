package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateConsentTokenSuccess(t *testing.T) {
	path, err := CreateConsentToken(OperationForceLock, "user confirmed the other deploy is stale and can be overridden")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	assert.FileExists(t, path)

	token, err := readConsentToken(path)
	require.NoError(t, err)
	assert.Equal(t, OperationForceLock, token.Operation)
	assert.Contains(t, token.Reason, "stale")
	assert.WithinDuration(t, time.Now(), token.CreatedAt, 5*time.Second)
}

func TestCreateConsentTokenInvalidOperation(t *testing.T) {
	_, err := CreateConsentToken("nuke-everything", "user said go ahead")
	assert.ErrorContains(t, err, "invalid operation")
}

func TestCreateConsentTokenReasonTooShort(t *testing.T) {
	_, err := CreateConsentToken(OperationAutoApprove, "yes")
	assert.ErrorContains(t, err, "at least 20 characters")
}

func TestValidateConsentNoAgent(t *testing.T) {
	ctx := Mock(t.Context(), "")
	err := ValidateConsent(ctx, OperationForceLock)
	assert.NoError(t, err)
}

func TestValidateConsentMissingToken(t *testing.T) {
	ctx := Mock(t.Context(), ClaudeCode)
	err := ValidateConsent(ctx, OperationForceLock)

	var consentErr *ConsentRequiredError
	assert.ErrorAs(t, err, &consentErr)
	assert.Equal(t, OperationForceLock, consentErr.Operation)
	assert.Contains(t, err.Error(), "explicit user consent")
}

func TestValidateConsentValidToken(t *testing.T) {
	path, err := CreateConsentToken(OperationAutoApprove, "user reviewed the plan and approved resource deletions")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	ctx := Mock(t.Context(), Cursor)
	ctx = env.Set(ctx, ConsentEnvVar, path)
	assert.NoError(t, ValidateConsent(ctx, OperationAutoApprove))
}

func TestValidateConsentWrongOperation(t *testing.T) {
	path, err := CreateConsentToken(OperationForceLock, "user confirmed the lock can be overridden safely")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	ctx := Mock(t.Context(), ClaudeCode)
	ctx = env.Set(ctx, ConsentEnvVar, path)
	err = ValidateConsent(ctx, OperationAutoApprove)
	assert.ErrorContains(t, err, "consent token is for")
}

func TestValidateConsentExpiredToken(t *testing.T) {
	path, err := CreateConsentToken(OperationForceDeploy, "user approved force deploy to override dashboard")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(path) })

	// Backdate the token file content.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	old := time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339)
	content := "operation: " + OperationForceDeploy + "\nreason: test\ncreated: " + old + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	_ = data

	ctx := Mock(t.Context(), Codex)
	ctx = env.Set(ctx, ConsentEnvVar, path)
	err = ValidateConsent(ctx, OperationForceDeploy)
	assert.ErrorContains(t, err, "expired")
}

func TestValidateConsentInvalidPath(t *testing.T) {
	ctx := Mock(t.Context(), ClaudeCode)
	ctx = env.Set(ctx, ConsentEnvVar, "/nonexistent/token")
	err := ValidateConsent(ctx, OperationForceLock)
	assert.ErrorContains(t, err, "invalid agent consent token")
}

func TestCleanExpiredTokens(t *testing.T) {
	dir := t.TempDir()

	// Create an old file.
	oldFile := filepath.Join(dir, "consent-old")
	require.NoError(t, os.WriteFile(oldFile, []byte("old"), 0o600))
	oldTime := time.Now().Add(-25 * time.Minute)
	require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))

	// Create a recent file.
	newFile := filepath.Join(dir, "consent-new")
	require.NoError(t, os.WriteFile(newFile, []byte("new"), 0o600))

	cleanExpiredTokens(dir)

	assert.NoFileExists(t, oldFile)
	assert.FileExists(t, newFile)
}
