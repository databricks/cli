package vscode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckIDECommand_Missing(t *testing.T) {
	// Override PATH to ensure commands are not found
	t.Setenv("PATH", t.TempDir())

	tests := []struct {
		name       string
		ide        string
		wantErrMsg string
	}{
		{
			name:       "missing vscode command",
			ide:        VSCodeOption,
			wantErrMsg: `"code" command not found on PATH`,
		},
		{
			name:       "missing cursor command",
			ide:        CursorOption,
			wantErrMsg: `"cursor" command not found on PATH`,
		},
		{
			name:       "vscode error contains install instructions",
			ide:        VSCodeOption,
			wantErrMsg: "https://code.visualstudio.com/",
		},
		{
			name:       "cursor error contains install instructions",
			ide:        CursorOption,
			wantErrMsg: "https://cursor.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckIDECommand(tt.ide)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestCheckIDECommand_Found(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)

	tests := []struct {
		name    string
		ide     string
		command string
	}{
		{
			name:    "vscode command found",
			ide:     VSCodeOption,
			command: "code",
		},
		{
			name:    "cursor command found",
			ide:     CursorOption,
			command: "cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake executable in the temp directory
			fakePath := filepath.Join(tmpDir, tt.command)
			err := os.WriteFile(fakePath, []byte("#!/bin/sh\n"), 0o755)
			require.NoError(t, err)

			err = CheckIDECommand(tt.ide)
			assert.NoError(t, err)
		})
	}
}
