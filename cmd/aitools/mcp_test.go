package aitools

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeDatabricksCfg writes a Databricks config file and points DATABRICKS_CONFIG_FILE at it.
// Profile selection variables are cleared so each test sets only what it needs.
func writeDatabricksCfg(t *testing.T, contents string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", path)
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
}

func TestMcpCommandIsHidden(t *testing.T) {
	mcp := NewMcpCmd()
	assert.True(t, mcp.Hidden)
	assert.NotNil(t, mcp.Commands())

	var names []string
	for _, c := range mcp.Commands() {
		names = append(names, c.Name())
	}
	assert.Equal(t, []string{"add", "list", "login", "logout", "remove", "serve"}, names)

	for _, c := range NewAitoolsCmd().Commands() {
		if c.Name() == "mcp" {
			assert.True(t, c.Hidden, "aitools mcp must stay hidden")
			return
		}
	}
	t.Fatal("aitools has no mcp command")
}

func TestResolveProfilePrecedence(t *testing.T) {
	ctx := t.Context()
	const twoProfiles = `[__settings__]
default_profile = dflt

[dflt]
host = https://dflt.test

[other]
host = https://other.test
`

	t.Run("flag wins over environment and default", func(t *testing.T) {
		writeDatabricksCfg(t, twoProfiles)
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "env")
		name, err := resolveProfile(ctx, "flag")
		require.NoError(t, err)
		assert.Equal(t, "flag", name)
	})

	t.Run("environment wins over default profile", func(t *testing.T) {
		writeDatabricksCfg(t, twoProfiles)
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "other")
		name, err := resolveProfile(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "other", name)
	})

	t.Run("default profile when nothing else is set", func(t *testing.T) {
		writeDatabricksCfg(t, twoProfiles)
		name, err := resolveProfile(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "dflt", name)
	})

	t.Run("only workspace profile", func(t *testing.T) {
		writeDatabricksCfg(t, `[solo]
host = https://solo.test
`)
		name, err := resolveProfile(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, "solo", name)
	})

	t.Run("several profiles and no default is an error listing them", func(t *testing.T) {
		writeDatabricksCfg(t, `[a]
host = https://a.test

[b]
host = https://b.test
`)
		_, err := resolveProfile(ctx, "")
		require.Error(t, err)
		assert.Equal(t, "several workspace profiles exist; pass --profile. Found: a, b", err.Error())
		assert.Equal(t, categoryProfileRequired, errorCategory(err))
	})

	t.Run("no workspace profiles is an error naming the file", func(t *testing.T) {
		writeDatabricksCfg(t, "")
		_, err := resolveProfile(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no workspace profiles in ")
		assert.Equal(t, categoryNoProfile, errorCategory(err))
		assert.Contains(t, err.Error(), "databricks auth login --host <url> --profile <name>")
	})
}

func TestErrorCategoryUsesTypedErrorsOnly(t *testing.T) {
	typed := failure(protos.AitoolsErrorCategoryUnknownAgent, nil, "unknown agent %q", "pi")
	assert.Equal(t, protos.AitoolsErrorCategoryUnknownAgent, errorCategory(typed))
	assert.Equal(t, "unknown agent \"pi\"", typed.Error())

	wrapped := failure(categoryServiceNotFound, errors.New("boom"), "%s", "boom")
	assert.Equal(t, categoryServiceNotFound, errorCategory(wrapped))

	configInvalid := &os.PathError{Op: "parse", Path: "x", Err: agentconfig.ErrConfigInvalid}
	assert.Equal(t, categoryAgentConfigInvalid, errorCategory(configInvalid))

	assert.Equal(t, protos.AitoolsErrorCategoryUncategorized, errorCategory(errors.New("some failure")))
}
