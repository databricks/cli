package lakebox

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// keyHash returns the identifier the lakebox SSH-keys API assigns to a
// public key. The algorithm is sha256("<type> <base64-blob>") truncated to
// the first 16 bytes and hex-encoded; the OpenSSH comment (anything after
// the second whitespace-separated token) is stripped before hashing, so
// registering the same key under different comments yields the same hash.
// Inputs that don't have a second token are hashed as-is.
//
// Useful for client-side checks like "is the local lakebox_rsa.pub already
// registered?" without a list call against /api/2.0/lakebox/ssh-keys.
func keyHash(publicKey string) string {
	canonical := publicKey
	if i := strings.IndexByte(publicKey, ' '); i >= 0 {
		if j := strings.IndexByte(publicKey[i+1:], ' '); j >= 0 {
			canonical = publicKey[:i+1+j]
		}
	}
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:16])
}
