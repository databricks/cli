package patchwheel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		a, b   string
		expect int
	}{
		{"1.2.10", "1.2.3", 1},
		{"1.2.3", "1.2.3", 0},
		{"1.2.3+2", "1.2.3", 1},
		{"10.0.0", "2.0.0", 1},
		{"1.2.3a", "1.2.3", 1},  // non-numeric suffix greater lexicographically
		{"1.2.3.1", "1.2.3", 1}, // leftover tokens make version greater
		{"1.2.3", "1.2.3.0", -1},
		{"0.0.1+20250604.74804", "0.0.1+20250604.74809", -1},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s_vs_%s", tc.a, tc.b), func(t *testing.T) {
			got := compareVersion(tc.a, tc.b)
			require.Equal(t, tc.expect, got)
		})

		// Mirror case
		t.Run(fmt.Sprintf("%s_vs_%s_mirror", tc.b, tc.a), func(t *testing.T) {
			got := compareVersion(tc.b, tc.a)
			require.Equal(t, -tc.expect, got)
		})
	}
}
