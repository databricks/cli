package serving_endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEndpointID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"31ee674b7e6643f6930d5e12148fbe6f", true},
		{"31EE674B7E6643F6930D5E12148FBE6F", false},
		{"31ee674b-7e66-43f6-930d-5e12148fbe6f", false},
		{"31ee674b7e6643f6930d5e12148fbe6", false},
		{"my-endpoint", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, isEndpointID(tt.in))
		})
	}
}
