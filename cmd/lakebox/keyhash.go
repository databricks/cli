package lakebox

import (
	"crypto/sha256"
	"encoding/hex"
)

// keyHash returns the identifier the lakebox SSH-keys API assigns to a
// public key. The algorithm is sha256("<type> <base64-blob>") truncated to
// the first 16 bytes and hex-encoded; the OpenSSH comment (anything after
// the second whitespace-separated token) is stripped before hashing, so
// registering the same key under different comments yields the same hash.
// Inputs that don't have a second token are hashed as-is.
//
// Useful for matching a locally-known key against entries in a
// GET /ssh-keys listing without sending the key contents back to the
// server.
func keyHash(publicKey string) string {
	// Slice off the OpenSSH comment by stopping at the second space.
	end := len(publicKey)
	spaces := 0
	for i, c := range publicKey {
		if c == ' ' {
			spaces++
			if spaces == 2 {
				end = i
				break
			}
		}
	}
	sum := sha256.Sum256([]byte(publicKey[:end]))
	return hex.EncodeToString(sum[:16])
}
