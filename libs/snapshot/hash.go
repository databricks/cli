package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashFromContent returns the SHA-256 hex digest of content.
func HashFromContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}
