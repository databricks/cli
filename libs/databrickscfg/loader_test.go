package databrickscfg

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestLoaderSkipsEmptyHost(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		Host: "",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
}

func TestLoaderSkipsExistingAuth(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		Host:  "https://foo",
		Token: "nonempty means pat auth",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
}

func TestLoaderSkipsExplicitAuthType(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "testdata/databrickscfg",
		Host:       "https://default",
		AuthType:   "azure-cli",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
	assert.Equal(t, "azure-cli", cfg.AuthType)
	assert.Empty(t, cfg.Profile)
	assert.Empty(t, cfg.Token)
}

func TestLoaderSkipsNonExistingConfigFile(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "idontexist",
		Host:       "https://default",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
	assert.Empty(t, cfg.Token)
}

func TestLoaderErrorsOnInvalidFile(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "profile/testdata/badcfg",
		Host:       "https://default",
	}

	err := cfg.EnsureResolved()
	assert.ErrorContains(t, err, "unclosed section: ")
}

func TestLoaderSkipsNoMatchingHost(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "profile/testdata/databrickscfg",
		Host:       "https://noneofthehostsmatch",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
	assert.Empty(t, cfg.Token)
}

func TestLoaderMatchingHost(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "profile/testdata/databrickscfg",
		Host:       "https://default",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
	assert.Equal(t, "default", cfg.Token)
	assert.Equal(t, "DEFAULT", cfg.Profile)
}

func TestLoaderMatchingHostWithQuery(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "profile/testdata/databrickscfg",
		Host:       "https://query/?foo=bar",
	}

	err := cfg.EnsureResolved()
	assert.NoError(t, err)
	assert.Equal(t, "query", cfg.Token)
	assert.Equal(t, "query", cfg.Profile)
}

func TestLoaderErrorsOnMultipleMatches(t *testing.T) {
	cfg := config.Config{
		Loaders: []config.Loader{
			ResolveProfileFromHost,
		},
		ConfigFile: "profile/testdata/databrickscfg",
		Host:       "https://foo/bar",
	}

	err := cfg.EnsureResolved()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "https://foo: multiple profiles matched: foo1, foo2")
}
