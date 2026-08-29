package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// keyHash returns the identifier the sandbox SSH-keys API assigns to a
// public key. The algorithm is sha256("<type> <base64-blob>") truncated to
// the first 16 bytes and hex-encoded; the OpenSSH comment (anything after
// the second whitespace-separated token) is stripped before hashing, so
// registering the same key under different comments yields the same hash.
// Leading and trailing whitespace are trimmed first — `.pub` files end
// with a newline that would otherwise be hashed in for comment-less keys.
func keyHash(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)
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
