package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePostgresName(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		projectID  string
		branchID   string
		endpointID string
		expectErr  bool
	}{
		{
			name:      "project",
			input:     "projects/my-project",
			projectID: "my-project",
		},
		{
			name:      "branch",
			input:     "projects/my-project/branches/my-branch",
			projectID: "my-project",
			branchID:  "my-branch",
		},
		{
			name:       "endpoint",
			input:      "projects/my-project/branches/my-branch/endpoints/my-endpoint",
			projectID:  "my-project",
			branchID:   "my-branch",
			endpointID: "my-endpoint",
		},
		{
			name:       "with hyphens and numbers",
			input:      "projects/my-app-123/branches/dev-branch/endpoints/primary-1",
			projectID:  "my-app-123",
			branchID:   "dev-branch",
			endpointID: "primary-1",
		},
		{
			name:      "empty",
			input:     "",
			expectErr: true,
		},
		{
			name:      "no prefix",
			input:     "my-project",
			expectErr: true,
		},
		{
			name:      "wrong prefix",
			input:     "project/my-project",
			expectErr: true,
		},
		{
			name:      "missing branch id",
			input:     "projects/my-project/branches/",
			expectErr: true,
		},
		{
			name:      "missing endpoint id",
			input:     "projects/my-project/branches/my-branch/endpoints/",
			expectErr: true,
		},
		{
			name:      "extra segments",
			input:     "projects/my-project/branches/my-branch/endpoints/my-endpoint/extra",
			expectErr: true,
		},
		{
			name:      "branches without branch",
			input:     "projects/my-project/branches",
			expectErr: true,
		},
		{
			name:      "endpoints without endpoint",
			input:     "projects/my-project/branches/my-branch/endpoints",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components, err := ParsePostgresName(tt.input)
			if tt.expectErr {
				assert.ErrorContains(t, err, "invalid postgres resource name format")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.projectID, components.ProjectID)
			assert.Equal(t, tt.branchID, components.BranchID)
			assert.Equal(t, tt.endpointID, components.EndpointID)
		})
	}
}
