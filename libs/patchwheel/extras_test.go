package patchwheel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitWheelExtras(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPath   string
		wantExtras string
	}{
		{
			name:       "basic extras",
			input:      "./dist/foo.whl[train]",
			wantPath:   "./dist/foo.whl",
			wantExtras: "[train]",
		},
		{
			name:       "multiple extras",
			input:      "./dist/foo.whl[train,test]",
			wantPath:   "./dist/foo.whl",
			wantExtras: "[train,test]",
		},
		{
			name:       "brackets in the middle are not extras",
			input:      "./dist/foo[12].whl",
			wantPath:   "./dist/foo[12].whl",
			wantExtras: "",
		},
		{
			name:       "brackets in the middle with trailing extras",
			input:      "./dist/foo[12].whl[train]",
			wantPath:   "./dist/foo[12].whl",
			wantExtras: "[train]",
		},
		{
			name:       "non-bracket suffix is not extras",
			input:      "./dist/foo.whl.bak",
			wantPath:   "./dist/foo.whl.bak",
			wantExtras: "",
		},
		{
			name:       "glob characters with extras",
			input:      "./dist/*.whl[train]",
			wantPath:   "./dist/*.whl",
			wantExtras: "[train]",
		},
		{
			name:       "no extras",
			input:      "./dist/foo.whl",
			wantPath:   "./dist/foo.whl",
			wantExtras: "",
		},
		{
			name:       "case insensitive extension",
			input:      "./dist/foo.WHL[train]",
			wantPath:   "./dist/foo.WHL",
			wantExtras: "[train]",
		},
		{
			name:       "empty extras",
			input:      "./dist/foo.whl[]",
			wantPath:   "./dist/foo.whl",
			wantExtras: "[]",
		},
		{
			name:       "unclosed bracket is not extras",
			input:      "./dist/foo.whl[train",
			wantPath:   "./dist/foo.whl[train",
			wantExtras: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, extras := SplitWheelExtras(tt.input)
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantExtras, extras)
		})
	}
}
