package databrickscfg

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOrCreate(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "databrickscfg")
	file, err := loadOrCreateConfigFile(path)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.FileExists(t, path)
}

func TestLoadOrCreate_NotAllowed(t *testing.T) {
	path := "/dev/databrickscfg"
	file, err := loadOrCreateConfigFile(path)
	assert.Error(t, err)
	assert.Nil(t, file)
	assert.NoFileExists(t, path)
}

func TestLoadOrCreate_Bad(t *testing.T) {
	path := "profile/testdata/badcfg"
	file, err := loadOrCreateConfigFile(path)
	assert.Error(t, err)
	assert.Nil(t, file)
}

func TestMatchOrCreateSection_Direct(t *testing.T) {
	cfg := &config.Config{
		Profile: "query",
	}
	file, err := loadOrCreateConfigFile("profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := context.Background()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "query", section.Name())
}

func TestMatchOrCreateSection_AccountID(t *testing.T) {
	cfg := &config.Config{
		AccountID: "abc",
	}
	file, err := loadOrCreateConfigFile("profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := context.Background()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "acc", section.Name())
}

func TestMatchOrCreateSection_NormalizeHost(t *testing.T) {
	cfg := &config.Config{
		Host: "https://query/?o=abracadabra",
	}
	file, err := loadOrCreateConfigFile("profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := context.Background()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "query", section.Name())
}

func TestMatchOrCreateSection_NoProfileOrHost(t *testing.T) {
	cfg := &config.Config{}
	file, err := loadOrCreateConfigFile("profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = matchOrCreateSection(ctx, file, cfg)
	assert.EqualError(t, err, "cannot create new profile: empty section name")
}

func TestMatchOrCreateSection_MultipleProfiles(t *testing.T) {
	cfg := &config.Config{
		Host: "https://foo",
	}
	file, err := loadOrCreateConfigFile("profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = matchOrCreateSection(ctx, file, cfg)
	assert.EqualError(t, err, "multiple profiles matched: foo1, foo2")
}

func TestMatchOrCreateSection_NewProfile(t *testing.T) {
	cfg := &config.Config{
		Host:    "https://bar",
		Profile: "delirium",
	}
	file, err := loadOrCreateConfigFile("profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := context.Background()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "delirium", section.Name())
}

func TestSaveToProfile_ErrorOnLoad(t *testing.T) {
	ctx := context.Background()
	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: "testdata/badcfg",
	})
	assert.Error(t, err)
}

func TestSaveToProfile_ErrorOnMatch(t *testing.T) {
	ctx := context.Background()
	err := SaveToProfile(ctx, &config.Config{
		Host: "https://foo",
	})
	assert.Error(t, err)
}

func TestSaveToProfile_NewFileWithoutDefault(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "abc",
		Host:       "https://foo",
		Token:      "xyz",
	})
	assert.NoError(t, err)
	assert.NoFileExists(t, path+".bak")

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t,
		`; The profile defined in the DEFAULT section is to be used as a fallback when no profile is explicitly specified.
[DEFAULT]

[abc]
host  = https://foo
token = xyz
`, string(contents))
}

func TestSaveToProfile_NewFileWithDefault(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "DEFAULT",
		Host:       "https://foo",
		Token:      "xyz",
	})
	assert.NoError(t, err)
	assert.NoFileExists(t, path+".bak")

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t,
		`[DEFAULT]
host  = https://foo
token = xyz
`, string(contents))
}

func TestSaveToProfile_ClearingPreviousProfile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "abc",
		Host:       "https://foo",
		Token:      "xyz",
	})
	assert.NoError(t, err)

	err = SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "bcd",
		Host:       "https://bar",
		Token:      "zyx",
	})
	assert.NoError(t, err)
	assert.FileExists(t, path+".bak")

	err = SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Host:       "https://foo",
		AuthType:   "databricks-cli",
	})
	assert.NoError(t, err)

	file, err := loadOrCreateConfigFile(path)
	assert.NoError(t, err)

	assert.Len(t, file.Sections(), 3)
	assert.True(t, file.HasSection("DEFAULT"))
	assert.True(t, file.HasSection("bcd"))
	assert.True(t, file.HasSection("bcd"))

	dlft, err := file.GetSection("DEFAULT")
	assert.NoError(t, err)
	assert.Empty(t, dlft.KeysHash())

	abc, err := file.GetSection("abc")
	assert.NoError(t, err)
	raw := abc.KeysHash()
	assert.Len(t, raw, 2)
	assert.Equal(t, "https://foo", raw["host"])
	assert.Equal(t, "databricks-cli", raw["auth_type"])
}
