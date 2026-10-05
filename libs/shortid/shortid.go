// Package shortid produces compact identifiers for the user agent and telemetry.
//
// An identifier is 40 bits encoded with the unpadded URL-safe base64 alphabet,
// which is 7 characters long. 40 bits is enough to correlate requests and events
// while keeping the user agent short. The alphabet (A-Z, a-z, 0-9, '-', '_') is
// accepted as a user agent value by the SDK.
package shortid

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const numBytes = 5

func encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b[:numBytes])
}

// New returns a random identifier.
func New() string {
	b := make([]byte, numBytes)
	// Read never returns an error since Go 1.24.
	_, _ = rand.Read(b)
	return encode(b)
}

// Hash returns the identifier derived from the SHA-256 hash of s.
func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return encode(h[:])
}
