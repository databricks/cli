package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractHostQueryParams(t *testing.T) {
	tests := []struct {
		name string
		host string
		want HostParams
	}{
		{
			name: "extract workspace_id from ?o=",
			host: "https://spog.example.com/?o=12345",
			want: HostParams{Host: "https://spog.example.com", WorkspaceID: "12345"},
		},
		{
			name: "extract both account_id and workspace_id",
			host: "https://spog.example.com/?o=12345&a=abc",
			want: HostParams{Host: "https://spog.example.com", WorkspaceID: "12345", AccountID: "abc"},
		},
		{
			name: "extract account_id from ?account_id=",
			host: "https://spog.example.com/?account_id=abc",
			want: HostParams{Host: "https://spog.example.com", AccountID: "abc"},
		},
		{
			name: "extract workspace_id from ?workspace_id=",
			host: "https://spog.example.com/?workspace_id=99999",
			want: HostParams{Host: "https://spog.example.com", WorkspaceID: "99999"},
		},
		{
			name: "no query params leaves host unchanged",
			host: "https://spog.example.com",
			want: HostParams{Host: "https://spog.example.com"},
		},
		{
			name: "non-numeric ?o= is skipped",
			host: "https://spog.example.com/?o=abc",
			want: HostParams{Host: "https://spog.example.com"},
		},
		{
			name: "non-numeric ?workspace_id= is skipped",
			host: "https://spog.example.com/?workspace_id=abc",
			want: HostParams{Host: "https://spog.example.com"},
		},
		{
			name: "invalid URL is left unchanged",
			host: "not a valid url ://???",
			want: HostParams{Host: "not a valid url ://???"},
		},
		{
			name: "empty host",
			host: "",
			want: HostParams{Host: ""},
		},
		{
			name: "trailing slash stripped",
			host: "https://spog.example.com/?o=12345",
			want: HostParams{Host: "https://spog.example.com", WorkspaceID: "12345"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHostQueryParams(tt.host)
			assert.Equal(t, tt.want, got)
		})
	}
}
