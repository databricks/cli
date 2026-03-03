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

func TestSaveToProfile_MergeSemantics(t *testing.T) {
	type saveOp struct {
		cfg       *config.Config
		clearKeys []string
	}

	testCases := []struct {
		name     string
		profile  string
		saves    []saveOp
		wantKeys map[string]string
	}{
		{
			name:    "preserves existing keys on merge",
			profile: "abc",
			saves: []saveOp{
				{cfg: &config.Config{Profile: "abc", Host: "https://foo", Token: "xyz"}},
				{cfg: &config.Config{Host: "https://foo", AuthType: "databricks-cli"}},
			},
			wantKeys: map[string]string{
				"host":      "https://foo",
				"auth_type": "databricks-cli",
				"token":     "xyz",
			},
		},
		{
			name:    "clear keys removes specified keys",
			profile: "abc",
			saves: []saveOp{
				{cfg: &config.Config{Profile: "abc", Host: "https://foo", Token: "xyz", ClusterID: "cluster-123"}},
				{cfg: &config.Config{Host: "https://foo", AuthType: "databricks-cli", ServerlessComputeID: "auto"}, clearKeys: []string{"token", "cluster_id"}},
			},
			wantKeys: map[string]string{
				"host":                  "https://foo",
				"auth_type":             "databricks-cli",
				"serverless_compute_id": "auto",
			},
		},
		{
			name:    "overwrites existing values",
			profile: "abc",
			saves: []saveOp{
				{cfg: &config.Config{Profile: "abc", Host: "https://old-host"}},
				{cfg: &config.Config{Profile: "abc", Host: "https://new-host"}},
			},
			wantKeys: map[string]string{
				"host": "https://new-host",
			},
		},
		{
			name:    "clear nonexistent key is noop",
			profile: "abc",
			saves: []saveOp{
				{cfg: &config.Config{Profile: "abc", Host: "https://foo"}},
				{cfg: &config.Config{Profile: "abc", Host: "https://foo", AuthType: "databricks-cli"}, clearKeys: []string{"token", "nonexistent_key"}},
			},
			wantKeys: map[string]string{
				"host":      "https://foo",
				"auth_type": "databricks-cli",
			},
		},
		{
			name:    "writes scopes as comma-separated",
			profile: "scoped",
			saves: []saveOp{
				{cfg: &config.Config{Profile: "scoped", Host: "https://myworkspace.cloud.databricks.com", AuthType: "databricks-cli", Scopes: []string{"jobs", "pipelines", "clusters"}}},
			},
			wantKeys: map[string]string{
				"host":      "https://myworkspace.cloud.databricks.com",
				"auth_type": "databricks-cli",
				"scopes":    "jobs,pipelines,clusters",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "databrickscfg")

			for _, save := range tc.saves {
				save.cfg.ConfigFile = path
				err := SaveToProfile(ctx, save.cfg, save.clearKeys...)
				require.NoError(t, err)
			}

			file, err := loadOrCreateConfigFile(path)
			require.NoError(t, err)

			section, err := file.GetSection(tc.profile)
			require.NoError(t, err)

			raw := section.KeysHash()
			assert.Len(t, raw, len(tc.wantKeys))
			for k, v := range tc.wantKeys {
				assert.Equal(t, v, raw[k], "key %s", k)
			}
		})
	}
}
