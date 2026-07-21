package genieclicmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInjectionCodex(t *testing.T) {
	inj, err := buildInjection(t.Context(), harnessCodex, "https://myworkspace.databricks.com", "prod")
	require.NoError(t, err)
	assert.Empty(t, inj.env)
	require.Len(t, inj.forwardArgs, 2)
	assert.Equal(t, "-c", inj.forwardArgs[0])

	// The prompt is delivered as a single quoted TOML basic string.
	value := inj.forwardArgs[1]
	assert.True(t, strings.HasPrefix(value, `developer_instructions="`))
	assert.True(t, strings.HasSuffix(value, `"`))
	body := strings.TrimPrefix(value, "developer_instructions=")
	assert.NotContains(t, body[1:len(body)-1], "\n")
	assert.Contains(t, body, `\n`)
	assert.Contains(t, value, "myworkspace.databricks.com")
}

func TestBuildInjectionOpenCode(t *testing.T) {
	home := t.TempDir()
	ctx := env.Set(t.Context(), "HOME", home)
	ctx = env.Set(ctx, "USERPROFILE", home)

	inj, err := buildInjection(ctx, harnessOpenCode, "https://myworkspace.databricks.com", "prod")
	require.NoError(t, err)
	assert.Empty(t, inj.forwardArgs)
	require.Len(t, inj.env, 1)

	// The env var carries JSON pointing at the managed prompt file, which must
	// exist and contain the rendered prompt.
	content := strings.TrimPrefix(inj.env[0], "OPENCODE_CONFIG_CONTENT=")
	assert.NotEqual(t, inj.env[0], content) // prefix was present

	var cfg struct {
		Instructions []string `json:"instructions"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &cfg))
	require.Len(t, cfg.Instructions, 1)

	data, err := os.ReadFile(cfg.Instructions[0])
	require.NoError(t, err)
	assert.Contains(t, string(data), "myworkspace.databricks.com")
	assert.Contains(t, string(data), "databricks experimental genie ask")
}

func TestBuildInjectionUnknownHarnessIsEmpty(t *testing.T) {
	inj, err := buildInjection(t.Context(), "gemini", "https://myworkspace.databricks.com", "prod")
	require.NoError(t, err)
	assert.Empty(t, inj.forwardArgs)
	assert.Empty(t, inj.env)
}
