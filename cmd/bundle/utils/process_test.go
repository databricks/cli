package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		current string
		want    bool
	}{
		{"state newer major", "1.0.0", "0.300.0", true},
		{"state newer minor", "0.301.0", "0.300.0", true},
		{"state newer patch", "0.300.1", "0.300.0", true},
		{"same version", "0.300.0", "0.300.0", false},
		{"state older", "0.299.0", "0.300.0", false},
		// A released CLI reading a state written by a dev build must not warn:
		// 0.0.0-dev sorts below every release.
		{"dev state, released current", "0.0.0-dev+abc123", "0.300.0", false},
		// A dev build reading a state written by a real release does warn, since
		// the release genuinely is newer.
		{"released state, dev current", "0.300.0", "0.0.0-dev+abc123", true},
		// A prerelease sorts below its own release per semver.
		{"prerelease below release", "0.300.0-rc1", "0.300.0", false},
		{"release above prerelease", "0.300.0", "0.300.0-rc1", true},
		// Missing or malformed data must never produce a warning.
		{"empty state version", "", "0.300.0", false},
		{"empty current version", "0.300.0", "", false},
		{"malformed state version", "not-a-version", "0.300.0", false},
		{"malformed current version", "0.300.0", "not-a-version", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isNewerVersion(tt.state, tt.current))
		})
	}
}
