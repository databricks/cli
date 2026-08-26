package client

import "testing"

func TestUsagePolicyMatches(t *testing.T) {
	tests := []struct {
		name      string
		stored    string
		requested string
		want      bool
	}{
		{name: "empty request matches any server", stored: "pol-1", requested: "", want: true},
		{name: "empty request matches server without policy", stored: "", requested: "", want: true},
		{name: "equal policies match", stored: "pol-1", requested: "pol-1", want: true},
		{name: "different policies do not match", stored: "pol-1", requested: "pol-2", want: false},
		{name: "request against server without policy does not match", stored: "", requested: "pol-1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usagePolicyMatches(tt.stored, tt.requested); got != tt.want {
				t.Errorf("usagePolicyMatches(%q, %q) = %v, want %v", tt.stored, tt.requested, got, tt.want)
			}
		})
	}
}

func TestKeepDetachedMatches(t *testing.T) {
	tests := []struct {
		name      string
		stored    int64
		requested int64
		want      bool
	}{
		{name: "empty request takes a lingering server", stored: 3600000, requested: 0, want: true},
		{name: "empty request takes a server that does not linger", stored: 0, requested: 0, want: true},
		{name: "equal durations match", stored: 3600000, requested: 3600000, want: true},
		{name: "different durations do not match", stored: 3600000, requested: 7200000, want: false},
		{name: "request against a server that does not linger does not match", stored: 0, requested: 3600000, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keepDetachedMatches(tt.stored, tt.requested); got != tt.want {
				t.Errorf("keepDetachedMatches(%d, %d) = %v, want %v", tt.stored, tt.requested, got, tt.want)
			}
		})
	}
}
