package databrickscfg

import (
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
	file, err := loadOrCreateConfigFile(t.Context(), path)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.FileExists(t, path)
}

func TestLoadOrCreate_NotAllowed(t *testing.T) {
	path := "/dev/databrickscfg"
	file, err := loadOrCreateConfigFile(t.Context(), path)
	assert.Error(t, err)
	assert.Nil(t, file)
	assert.NoFileExists(t, path)
}

func TestLoadOrCreate_Bad(t *testing.T) {
	path := "profile/testdata/badcfg"
	file, err := loadOrCreateConfigFile(t.Context(), path)
	assert.Error(t, err)
	assert.Nil(t, file)
}

func TestMatchOrCreateSection_Direct(t *testing.T) {
	cfg := &config.Config{
		Profile: "query",
	}
	file, err := loadOrCreateConfigFile(t.Context(), "profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := t.Context()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "query", section.Name())
}

func TestMatchOrCreateSection_AccountID(t *testing.T) {
	cfg := &config.Config{
		AccountID: "abc",
	}
	file, err := loadOrCreateConfigFile(t.Context(), "profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := t.Context()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "acc", section.Name())
}

func TestMatchOrCreateSection_NormalizeHost(t *testing.T) {
	cfg := &config.Config{
		Host: "https://query/?o=abracadabra",
	}
	file, err := loadOrCreateConfigFile(t.Context(), "profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := t.Context()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "query", section.Name())
}

func TestMatchOrCreateSection_NoProfileOrHost(t *testing.T) {
	cfg := &config.Config{}
	file, err := loadOrCreateConfigFile(t.Context(), "profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := t.Context()
	_, err = matchOrCreateSection(ctx, file, cfg)
	assert.EqualError(t, err, "cannot create new profile: empty section name")
}

func TestMatchOrCreateSection_MultipleProfiles(t *testing.T) {
	cfg := &config.Config{
		Host: "https://foo",
	}
	file, err := loadOrCreateConfigFile(t.Context(), "profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := t.Context()
	_, err = matchOrCreateSection(ctx, file, cfg)
	assert.EqualError(t, err, "multiple profiles matched: foo1, foo2")
}

func TestMatchOrCreateSection_NewProfile(t *testing.T) {
	cfg := &config.Config{
		Host:    "https://bar",
		Profile: "delirium",
	}
	file, err := loadOrCreateConfigFile(t.Context(), "profile/testdata/databrickscfg")
	assert.NoError(t, err)

	ctx := t.Context()
	section, err := matchOrCreateSection(ctx, file, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, section)
	assert.Equal(t, "delirium", section.Name())
}

func TestSaveToProfile_ErrorOnLoad(t *testing.T) {
	ctx := t.Context()
	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: "testdata/badcfg",
	})
	assert.Error(t, err)
}

func TestSaveToProfile_ErrorOnMatch(t *testing.T) {
	ctx := t.Context()
	err := SaveToProfile(ctx, &config.Config{
		Host: "https://foo",
	})
	assert.Error(t, err)
}

func TestSaveToProfile_NewFileWithoutDefault(t *testing.T) {
	ctx := t.Context()
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
	ctx := t.Context()
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

func TestGetDefaultProfile(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "explicit default_profile setting",
			content: "[databricks-cli-settings]\ndefault_profile = my-workspace\n\n[my-workspace]\nhost = https://abc\n",
			want:    "my-workspace",
		},
		{
			name:    "single profile fallback",
			content: "[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
		{
			name:    "multiple profiles no default",
			content: "[profile1]\nhost = https://abc\n\n[profile2]\nhost = https://def\n",
			want:    "",
		},
		{
			name:    "multiple profiles with DEFAULT fallback",
			content: "[DEFAULT]\nhost = https://abc\n\n[profile2]\nhost = https://def\n",
			want:    "DEFAULT",
		},
		{
			name:    "settings section without key single profile",
			content: "[databricks-cli-settings]\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
		{
			name:    "empty config file",
			content: "",
			want:    "",
		},
		{
			name:    "settings section is not counted as a profile",
			content: "[databricks-cli-settings]\nsome_key = value\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
		{
			name:    "section without host is not a profile",
			content: "[no-host]\naccount_id = abc\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "databrickscfg")
			err := os.WriteFile(path, []byte(tc.content), 0o600)
			require.NoError(t, err)

			got, err := GetDefaultProfile(context.Background(), path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetDefaultProfile_NoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databrickscfg")
	got, err := GetDefaultProfile(context.Background(), path)
	require.NoError(t, err)
	assert.Equal(t, "", got)
	// Verify the file was NOT created as a side effect.
	assert.NoFileExists(t, path)
}

func TestSetDefaultProfile(t *testing.T) {
	testCases := []struct {
		name    string
		initial string
		profile string
		wantKey string
	}{
		{
			name:    "creates section and key",
			initial: "[profile1]\nhost = https://abc\n",
			profile: "profile1",
			wantKey: "profile1",
		},
		{
			name:    "updates existing key",
			initial: "[databricks-cli-settings]\ndefault_profile = old-profile\n\n[profile1]\nhost = https://abc\n",
			profile: "new-profile",
			wantKey: "new-profile",
		},
		{
			name:    "creates section in empty file",
			initial: "",
			profile: "my-workspace",
			wantKey: "my-workspace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "databrickscfg")
			err := os.WriteFile(path, []byte(tc.initial), 0o600)
			require.NoError(t, err)

			err = SetDefaultProfile(ctx, tc.profile, path)
			require.NoError(t, err)

			got, err := GetDefaultProfile(ctx, path)
			require.NoError(t, err)
			assert.Equal(t, tc.wantKey, got)
		})
	}
}

func TestSetDefaultProfile_RoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	// Start with a profile.
	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "my-workspace",
		Host:       "https://abc.cloud.databricks.com",
		Token:      "xyz",
	})
	require.NoError(t, err)

	// Set it as default.
	err = SetDefaultProfile(ctx, "my-workspace", path)
	require.NoError(t, err)

	// Read it back.
	got, err := GetDefaultProfile(ctx, path)
	require.NoError(t, err)
	assert.Equal(t, "my-workspace", got)

	// Verify the profile section is still intact.
	file, err := loadOrCreateConfigFile(ctx, path)
	require.NoError(t, err)
	section, err := file.GetSection("my-workspace")
	require.NoError(t, err)
	assert.Equal(t, "https://abc.cloud.databricks.com", section.Key("host").String())
	assert.Equal(t, "xyz", section.Key("token").String())
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
			ctx := t.Context()
			path := filepath.Join(t.TempDir(), "databrickscfg")

			for _, save := range tc.saves {
				save.cfg.ConfigFile = path
				err := SaveToProfile(ctx, save.cfg, save.clearKeys...)
				require.NoError(t, err)
			}

			file, err := loadOrCreateConfigFile(t.Context(), path)
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
