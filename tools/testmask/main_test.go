package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyRules(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		packages   []string
		acceptance []string
	}{
		{
			name: "experimental_subdir",
			files: []string{
				"experimental/ssh/main.go",
				"experimental/ssh/lib/server.go",
			},
			packages:   []string{"experimental/ssh/..."},
			acceptance: []string{},
		},
		{
			name: "multiple_experimental_subdirs",
			files: []string{
				"experimental/ssh/main.go",
				"experimental/aitools/server.go",
			},
			packages: []string{
				"experimental/...",
			},
			acceptance: []string{},
		},
		{
			name: "non_experimental",
			files: []string{
				"bundle/config.go",
				"cmd/bundle/deploy.go",
			},
			packages:   []string{},
			acceptance: []string{},
		},
		{
			name: "mixed_experimental_and_other",
			files: []string{
				"experimental/ssh/main.go",
				"go.mod",
			},
			packages:   []string{},
			acceptance: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages, acceptance := applyRules(tt.files)
			assert.Equal(t, tt.packages, packages)
			assert.Equal(t, tt.acceptance, acceptance)
		})
	}
}
