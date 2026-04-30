package postgrescmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateTargeting(t *testing.T) {
	tests := []struct {
		name    string
		flags   targetingFlags
		wantErr string
	}{
		{
			name:    "neither form",
			flags:   targetingFlags{},
			wantErr: "must specify --target or --project",
		},
		{
			name: "only target",
			flags: targetingFlags{
				target: "projects/foo",
			},
		},
		{
			name: "only project",
			flags: targetingFlags{
				project: "foo",
			},
		},
		{
			name: "project and branch",
			flags: targetingFlags{
				project: "foo",
				branch:  "main",
			},
		},
		{
			name: "project, branch, endpoint",
			flags: targetingFlags{
				project:  "foo",
				branch:   "main",
				endpoint: "primary",
			},
		},
		{
			name: "target and project both set",
			flags: targetingFlags{
				target:  "projects/foo",
				project: "foo",
			},
			wantErr: "mutually exclusive",
		},
		{
			name: "branch without project",
			flags: targetingFlags{
				branch: "main",
			},
			wantErr: "--project is required when using --branch or --endpoint",
		},
		{
			name: "endpoint without project",
			flags: targetingFlags{
				endpoint: "primary",
			},
			wantErr: "--project is required when using --branch or --endpoint",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTargeting(tc.flags)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
