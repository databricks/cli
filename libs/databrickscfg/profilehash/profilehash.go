package profilehash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// Compute hashes every field in the simplified profile representation.
func Compute(p profile.Profile) (string, error) {
	normalized := p
	normalized.Scopes = normalizeScopes(normalized.Scopes)

	// Marshal the whole simplified profile so newly added profile fields are
	// included automatically. Only the code constructing Profile decides which
	// configuration fields belong in the fingerprint.
	serialized, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(serialized)

	return hex.EncodeToString(sum[:]), nil
}

// A stored profile preserves the order in which its scopes were written, while
// resolving a configuration sorts and removes duplicate scopes. Normalize the
// stored value in the same way so both forms produce the same fingerprint.
func normalizeScopes(value string) string {
	scopes := strings.Split(value, ",")
	for i := range scopes {
		scopes[i] = strings.TrimSpace(scopes[i])
	}
	scopes = slices.DeleteFunc(scopes, func(scope string) bool {
		return scope == ""
	})
	slices.Sort(scopes)
	scopes = slices.Compact(scopes)

	return strings.Join(scopes, ",")
}
