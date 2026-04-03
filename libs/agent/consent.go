package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/env"
)

// ConsentEnvVar is the environment variable that agents must set to a valid
// consent token path before using gated flags like --force-lock or --auto-approve.
const ConsentEnvVar = "DATABRICKS_CLI_AGENT_CONSENT"

// consentTokenDir is the directory where consent tokens are stored.
const consentTokenDir = "databricks-agent-consent"

// consentTokenExpiry is how long a consent token remains valid.
const consentTokenExpiry = 10 * time.Minute

// minReasonLength is the minimum length for consent reasons to prevent
// agents from using trivial reasons like "yes" or "ok".
const minReasonLength = 20

// Operations that can be consented to.
const (
	OperationForceLock   = "force-lock"
	OperationAutoApprove = "auto-approve"
	OperationForceDeploy = "force-deploy"
)

// ValidOperations lists all valid consent operations.
var ValidOperations = []string{
	OperationForceLock,
	OperationAutoApprove,
	OperationForceDeploy,
}

// ConsentToken represents a validated consent token.
type ConsentToken struct {
	Operation string
	Reason    string
	CreatedAt time.Time
}

// CreateConsentToken writes a consent token file and returns its path.
func CreateConsentToken(operation, reason string) (string, error) {
	if err := validateOperation(operation); err != nil {
		return "", err
	}
	if len(reason) < minReasonLength {
		return "", fmt.Errorf("consent reason must be at least %d characters to ensure meaningful justification, got %d", minReasonLength, len(reason))
	}

	dir := filepath.Join(os.TempDir(), consentTokenDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create consent token directory: %w", err)
	}

	// Clean up expired tokens on each creation.
	cleanExpiredTokens(dir)

	tokenID := make([]byte, 16)
	_, err := rand.Read(tokenID)
	if err != nil {
		return "", fmt.Errorf("failed to generate token ID: %w", err)
	}

	filename := fmt.Sprintf("consent-%s-%s", operation, hex.EncodeToString(tokenID))
	tokenPath := filepath.Join(dir, filename)

	content := fmt.Sprintf("operation: %s\nreason: %s\ncreated: %s\n",
		operation, reason, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(tokenPath, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("failed to write consent token: %w", err)
	}

	return tokenPath, nil
}

// ValidateConsent checks whether the agent has valid consent for the given operation.
// It reads the token path from the DATABRICKS_CLI_AGENT_CONSENT environment variable.
// Returns nil if no agent is detected (non-agent callers are not gated).
// Returns an error if agent is detected but consent is missing or invalid.
func ValidateConsent(ctx context.Context, operation string) error {
	// Non-agent callers are not gated.
	if Product(ctx) == "" {
		return nil
	}

	tokenPath := env.Get(ctx, ConsentEnvVar)
	if tokenPath == "" {
		return &ConsentRequiredError{Operation: operation}
	}

	token, err := readConsentToken(tokenPath)
	if err != nil {
		return fmt.Errorf("invalid agent consent token: %w", err)
	}

	if time.Since(token.CreatedAt) > consentTokenExpiry {
		return fmt.Errorf("agent consent token has expired (created %s ago, max %s). Run `databricks agent consent` again",
			time.Since(token.CreatedAt).Round(time.Second), consentTokenExpiry)
	}

	if token.Operation != operation {
		return fmt.Errorf("agent consent token is for %q, but %q is required", token.Operation, operation)
	}

	return nil
}

// ConsentRequiredError is returned when an agent attempts a gated operation
// without providing consent.
type ConsentRequiredError struct {
	Operation string
}

func (e *ConsentRequiredError) Error() string {
	return fmt.Sprintf(
		"this operation requires explicit user consent when run by an AI agent.\n\n"+
			"AI agents must not automatically retry with this flag. Instead:\n"+
			"1. Present this error to the user and explain what the flag does\n"+
			"2. If the user approves, run: databricks agent consent --operation %s --reason \"<why the user approved>\"\n"+
			"3. Set the %s environment variable to the output token path\n"+
			"4. Retry the command",
		e.Operation, ConsentEnvVar)
}

func readConsentToken(path string) (*ConsentToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read token file %s: %w", path, err)
	}

	token := &ConsentToken{}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "operation: ") {
			token.Operation = strings.TrimPrefix(line, "operation: ")
		} else if strings.HasPrefix(line, "reason: ") {
			token.Reason = strings.TrimPrefix(line, "reason: ")
		} else if strings.HasPrefix(line, "created: ") {
			t, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "created: "))
			if err != nil {
				return nil, fmt.Errorf("invalid timestamp in token: %w", err)
			}
			token.CreatedAt = t
		}
	}

	if token.Operation == "" {
		return nil, fmt.Errorf("token file is missing operation field")
	}

	return token, nil
}

func validateOperation(operation string) error {
	for _, op := range ValidOperations {
		if op == operation {
			return nil
		}
	}
	return fmt.Errorf("invalid operation %q, must be one of: %s", operation, strings.Join(ValidOperations, ", "))
}

func cleanExpiredTokens(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > consentTokenExpiry*2 {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
