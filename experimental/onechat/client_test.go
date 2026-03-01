package onechat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequest_WithoutWarehouse(t *testing.T) {
	req := BuildRequest("What tables exist?", "")

	require.Len(t, req.Input, 1)
	assert.Equal(t, "message", req.Input[0].Type)
	assert.Equal(t, "user", req.Input[0].Role)
	require.Len(t, req.Input[0].Content, 1)
	assert.Equal(t, "input_text", req.Input[0].Content[0].Type)
	assert.Equal(t, "What tables exist?", req.Input[0].Content[0].Text)
	assert.Empty(t, req.WarehouseID)
}

func TestBuildRequest_WithWarehouse(t *testing.T) {
	req := BuildRequest("What tables exist?", "abc123")

	require.Len(t, req.Input, 1)
	assert.Equal(t, "abc123", req.WarehouseID)
}
