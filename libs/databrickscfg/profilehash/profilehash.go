package profilehash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// Compute hashes every field in the simplified profile representation.
func Compute(p profile.Profile) (string, error) {
	// Marshal the whole simplified profile so newly added profile fields are
	// included automatically. Only the code constructing Profile decides which
	// configuration fields belong in the fingerprint.
	serialized, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(serialized)

	return hex.EncodeToString(sum[:]), nil
}
