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
		stored    bool
		requested bool
		want      bool
	}{
		{name: "no request takes a server that holds the run open", stored: true, requested: false, want: true},
		{name: "no request takes a server that does not", stored: false, requested: false, want: true},
		{name: "request matches a server that holds the run open", stored: true, requested: true, want: true},
		{name: "request against a server that does not does not match", stored: false, requested: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := keepDetachedMatches(tt.stored, tt.requested); got != tt.want {
				t.Errorf("keepDetachedMatches(%v, %v) = %v, want %v", tt.stored, tt.requested, got, tt.want)
			}
		})
	}
}
