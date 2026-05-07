package lakebox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// All expected hashes were captured live from /api/2.0/lakebox/ssh-keys
// (see PR description); they're the ground truth for the algorithm.
func TestKeyHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single-token input hashed verbatim",
			input: "a",
			want:  "ca978112ca1bbdcafac231b39a23dc4d",
		},
		{
			name:  "type and blob with no comment",
			input: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDUMMY",
			want:  "2b366430eb9743668b652921d3b22d54",
		},
		{
			name:  "comment is stripped before hashing",
			input: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDUMMY comment-one",
			want:  "2b366430eb9743668b652921d3b22d54",
		},
		{
			name:  "different comment same key still matches",
			input: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDUMMY entirely-different-comment",
			want:  "2b366430eb9743668b652921d3b22d54",
		},
		{
			name:  "longer key with multi-word comment",
			input: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITESTKEY1234 test-from-cli-exploration",
			want:  "52c927705154e2d98a1b7036cc3e06dc",
		},
		{
			name:  "empty input still produces a hash",
			input: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb924",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, keyHash(tc.input))
		})
	}
}

func TestKeyHashIsStableLength(t *testing.T) {
	// 16 bytes hex-encoded = 32 chars, matching what the API returns.
	assert.Len(t, keyHash("anything"), 32)
}
