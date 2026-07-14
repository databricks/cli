package auth_test

import (
	"testing"

	"github.com/databricks/cli/libs/auth"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestWorkspaceIDHeaders(t *testing.T) {
	tests := []struct {
		name string
		cfg  *sdkconfig.Config
		want map[string]string
	}{
		{
			name: "configured numeric workspace ID",
			cfg:  &sdkconfig.Config{WorkspaceID: "12345"},
			want: map[string]string{"X-Databricks-Workspace-Id": "12345"},
		},
		{
			name: "configured connection-id-style workspace ID",
			cfg:  &sdkconfig.Config{WorkspaceID: "123e4567-e89b-12d3-a456-426614174000"},
			want: map[string]string{"X-Databricks-Workspace-Id": "123e4567-e89b-12d3-a456-426614174000"},
		},
		{
			name: "empty workspace ID returns nil",
			cfg:  &sdkconfig.Config{WorkspaceID: ""},
			want: nil,
		},
		{
			name: `"none" sentinel returns nil`,
			cfg:  &sdkconfig.Config{WorkspaceID: auth.WorkspaceIDNone},
			want: nil,
		},
		{
			name: "nil config returns nil",
			cfg:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, auth.WorkspaceIDHeaders(tt.cfg))
		})
	}
}
