// Package shortid produces compact identifiers for the user agent and telemetry.
//
// An identifier is 64 bits encoded in base62 (0-9, A-Z, a-z), which is 11
// characters long, about a third of a UUID. Plain alphanumerics avoid '-' and
// '_', which read as separators in user agent and log values.
package shortid

import (
	"crypto/rand"
	"crypto/sha256"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	numBytes = 8

	// Length is fixed: 62^11 > 2^64, so every 64-bit value fits in 11 digits.
	length = 11
)

func encode(b []byte) string {
	var n uint64
	for _, c := range b[:numBytes] {
		n = n<<8 | uint64(c)
	}
	out := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		out[i] = alphabet[n%uint64(len(alphabet))]
		n /= uint64(len(alphabet))
	}
	return string(out)
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
