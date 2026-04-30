package psql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseResourcePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		project  string
		branch   string
		endpoint string
		wantErr  string
	}{
		{
			name:    "project only",
			input:   "projects/my-project",
			project: "my-project",
		},
		{
			name:    "project and branch",
			input:   "projects/my-project/branches/main",
			project: "my-project",
			branch:  "main",
		},
		{
			name:     "full path",
			input:    "projects/my-project/branches/main/endpoints/primary",
			project:  "my-project",
			branch:   "main",
			endpoint: "primary",
		},
		{
			name:    "missing project ID",
			input:   "projects/",
			wantErr: "missing project ID",
		},
		{
			name:    "missing branch ID",
			input:   "projects/my-project/branches/",
			wantErr: "missing branch ID",
		},
		{
			name:    "missing endpoint ID",
			input:   "projects/my-project/branches/main/endpoints/",
			wantErr: "missing endpoint ID",
		},
		{
			name:    "invalid segment after project",
			input:   "projects/my-project/invalid/foo",
			wantErr: "expected 'branches' after project",
		},
		{
			name:    "invalid segment after branch",
			input:   "projects/my-project/branches/main/invalid/foo",
			wantErr: "expected 'endpoints' after branch",
		},
		{
			name:    "not a projects path",
			input:   "something/else",
			wantErr: "invalid resource path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			project, branch, endpoint, err := parseResourcePath(tc.input)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.project, project)
			assert.Equal(t, tc.branch, branch)
			assert.Equal(t, tc.endpoint, endpoint)
		})
	}
}
