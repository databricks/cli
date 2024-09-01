package auth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
)

func TestLogout_ClearConfigFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "abc",
		Host:       "https://foo",
		Token:      "xyz",
	})
	require.NoError(t, err)
	iniFile, err := config.LoadFile(path)
	require.NoError(t, err)
	logout := &LogoutSession{
		Profile: "abc",
		File:    *iniFile,
	}
	section, err := logout.File.GetSection("abc")
	assert.NoError(t, err)
	sectionMap := section.KeysHash()
	err = logout.clearConfigFile(ctx, sectionMap)
	assert.NoError(t, err)

	iniFile, err = config.LoadFile(path)
	require.NoError(t, err)
	assert.Len(t, iniFile.Sections(), 2)
	assert.True(t, iniFile.HasSection("DEFAULT"))
	assert.True(t, iniFile.HasSection("abc"))

	abc, err := iniFile.GetSection("abc")
	assert.NoError(t, err)
	raw := abc.KeysHash()
	assert.Len(t, raw, 1)
	assert.Equal(t, "https://foo", raw["host"])
}
