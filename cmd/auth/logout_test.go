package auth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/libs/auth"
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
	logout := &logoutSession{
		profile: "abc",
		file:    *iniFile,
	}
	section, err := logout.file.GetSection("abc")
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

func TestLogout_setHostAndAccountIdFromProfile(t *testing.T) {
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
	logout := &logoutSession{
		profile:        "abc",
		file:           *iniFile,
		persistentAuth: &auth.PersistentAuth{},
	}
	err = logout.setHostAndAccountIdFromProfile()
	assert.NoError(t, err)
	assert.Equal(t, logout.persistentAuth.Host, "https://foo")
	assert.Empty(t, logout.persistentAuth.AccountID)
}

func TestLogout_getConfigSectionMap(t *testing.T) {
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
	logout := &logoutSession{
		profile:        "abc",
		file:           *iniFile,
		persistentAuth: &auth.PersistentAuth{},
	}
	configSectionMap, err := logout.getConfigSectionMap()
	assert.NoError(t, err)
	assert.Equal(t, configSectionMap["host"], "https://foo")
	assert.Equal(t, configSectionMap["token"], "xyz")
}
