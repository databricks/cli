package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAutoscalingPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AutoscalingSpec
		wantErr string
	}{
		{
			name:  "project only",
			input: "projects/my-project",
			want:  AutoscalingSpec{ProjectID: "my-project"},
		},
		{
			name:  "project and branch",
			input: "projects/my-project/branches/main",
			want:  AutoscalingSpec{ProjectID: "my-project", BranchID: "main"},
		},
		{
			name:  "full path",
			input: "projects/my-project/branches/main/endpoints/primary",
			want:  AutoscalingSpec{ProjectID: "my-project", BranchID: "main", EndpointID: "primary"},
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
		{
			name:    "trailing components after endpoint",
			input:   "projects/foo/branches/bar/endpoints/baz/extra",
			wantErr: "trailing components after endpoint",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAutoscalingPath(tc.input)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractID(t *testing.T) {
	assert.Equal(t, "bar", ExtractID("projects/foo/branches/bar", "branches"))
	assert.Equal(t, "foo", ExtractID("projects/foo", "projects"))
	assert.Equal(t, "baz", ExtractID("projects/foo/branches/bar/endpoints/baz", "endpoints"))
	assert.Equal(t, "no-component", ExtractID("no-component", "missing"))
}

func TestIsAutoscalingPath(t *testing.T) {
	assert.True(t, IsAutoscalingPath("projects/foo"))
	assert.True(t, IsAutoscalingPath("projects/foo/branches/bar"))
	assert.False(t, IsAutoscalingPath("my-instance"))
	assert.False(t, IsAutoscalingPath(""))
	assert.False(t, IsAutoscalingPath("projects"))
}

func TestAmbiguousErrorMessage(t *testing.T) {
	t.Run("with parent", func(t *testing.T) {
		err := &AmbiguousError{
			Kind:     "branch",
			Parent:   "projects/foo",
			FlagHint: "--branch",
			Choices: []Choice{
				{ID: "main", DisplayName: "main"},
				{ID: "feature-x", DisplayName: "feature-x"},
			},
		}
		assert.Equal(t,
			"multiple branches found in projects/foo; specify --branch:\n  - main\n  - feature-x",
			err.Error(),
		)
	})

	t.Run("without parent", func(t *testing.T) {
		err := &AmbiguousError{
			Kind:     "project",
			FlagHint: "--project",
			Choices: []Choice{
				{ID: "alpha", DisplayName: "Alpha Project"},
				{ID: "beta", DisplayName: "beta"},
			},
		}
		assert.Equal(t,
			"multiple projects found; specify --project:\n  - alpha (Alpha Project)\n  - beta",
			err.Error(),
		)
	})

	t.Run("errors.As", func(t *testing.T) {
		var amb *AmbiguousError
		err := error(&AmbiguousError{Kind: "endpoint", FlagHint: "--endpoint"})
		assert.ErrorAs(t, err, &amb)
		assert.Equal(t, "endpoint", amb.Kind)
	})
}
