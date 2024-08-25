package auth

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

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
	logout := &Logout{
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

type tokenCacheMock struct {
	store     func(key string, t *oauth2.Token) error
	lookup    func(key string) (*oauth2.Token, error)
	deleteKey func(key string) error
}

func (m *tokenCacheMock) Store(key string, t *oauth2.Token) error {
	if m.store == nil {
		panic("no store mock")
	}
	return m.store(key, t)
}

func (m *tokenCacheMock) Lookup(key string) (*oauth2.Token, error) {
	if m.lookup == nil {
		panic("no lookup mock")
	}
	return m.lookup(key)
}

func (m *tokenCacheMock) DeleteKey(key string) error {
	if m.deleteKey == nil {
		panic("no deleteKey mock")
	}
	return m.deleteKey(key)
}

func TestLogout_ClearTokenCache(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	err := databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "abc",
		Host:       "https://foo",
		AuthType:   "databricks-cli",
	})
	require.NoError(t, err)
	iniFile, err := config.LoadFile(path)
	require.NoError(t, err)
	logout := &Logout{
		Profile: "abc",
		File:    *iniFile,
		Cache: &tokenCacheMock{
			deleteKey: func(key string) error {
				assert.Equal(t, "https://foo", key)
				return nil
			},
			lookup: func(key string) (*oauth2.Token, error) {
				assert.Equal(t, "https://foo", key)
				return &oauth2.Token{}, fmt.Errorf("No token found")
			},
		},
	}
	sectionMap, err := logout.getSetionMap()
	assert.NoError(t, err)
	err = logout.clearTokenCache(sectionMap["host"])
	assert.NoError(t, err)
	_, err = logout.Cache.Lookup(sectionMap["host"])
	assert.Error(t, err)
}
