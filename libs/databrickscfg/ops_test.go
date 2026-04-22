package databrickscfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ini.v1"
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

[__settings__]
default_profile = abc
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

[__settings__]
default_profile = DEFAULT
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
			content: "[__settings__]\ndefault_profile = my-workspace\n\n[my-workspace]\nhost = https://abc\n",
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
			content: "[__settings__]\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
		{
			name:    "empty config file",
			content: "",
			want:    "",
		},
		{
			name:    "settings section is not counted as a profile",
			content: "[__settings__]\nsome_key = value\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
		{
			name:    "section without host is not a profile",
			content: "[no-host]\naccount_id = abc\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
		{
			name:    "self-referencing __settings__ falls through to single profile",
			content: "[__settings__]\ndefault_profile = __settings__\n\n[profile1]\nhost = https://abc\n",
			want:    "profile1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "databrickscfg")
			err := os.WriteFile(path, []byte(tc.content), 0o600)
			require.NoError(t, err)

			got, err := GetDefaultProfile(t.Context(), path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetDefaultProfile_NoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databrickscfg")
	got, err := GetDefaultProfile(t.Context(), path)
	require.NoError(t, err)
	assert.Equal(t, "", got)
	// Verify the file was NOT created as a side effect.
	assert.NoFileExists(t, path)
}

func TestGetConfiguredDefaultProfile(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "explicit default_profile setting",
			content: "[__settings__]\ndefault_profile = my-workspace\n\n[my-workspace]\nhost = https://abc\n",
			want:    "my-workspace",
		},
		{
			name:    "single profile fallback is ignored",
			content: "[profile1]\nhost = https://abc\n",
			want:    "",
		},
		{
			name:    "DEFAULT fallback is ignored",
			content: "[DEFAULT]\nhost = https://abc\n\n[profile2]\nhost = https://def\n",
			want:    "",
		},
		{
			name:    "settings section without key",
			content: "[__settings__]\n\n[profile1]\nhost = https://abc\n",
			want:    "",
		},
		{
			name:    "self-referencing __settings__ is ignored",
			content: "[__settings__]\ndefault_profile = __settings__\n\n[profile1]\nhost = https://abc\n",
			want:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "databrickscfg")
			err := os.WriteFile(path, []byte(tc.content), 0o600)
			require.NoError(t, err)

			got, err := GetConfiguredDefaultProfile(t.Context(), path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetConfiguredDefaultProfile_NoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databrickscfg")
	got, err := GetConfiguredDefaultProfile(t.Context(), path)
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
			initial: "[__settings__]\ndefault_profile = old-profile\n\n[profile1]\nhost = https://abc\n",
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
			ctx := t.Context()
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
	ctx := t.Context()
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

func TestSaveToProfile_RejectsReservedProfileName(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "databrickscfg")

	err := SaveToProfile(ctx, &config.Config{
		ConfigFile: path,
		Profile:    "__settings__",
		Host:       "https://abc.cloud.databricks.com",
		Token:      "token",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved for internal use")
}

func TestSetDefaultProfile_RejectsReservedProfileName(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "databrickscfg")
	err := os.WriteFile(path, []byte("[profile1]\nhost = https://abc\n"), 0o600)
	require.NoError(t, err)

	err = SetDefaultProfile(ctx, "__settings__", path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved for internal use")
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

func TestDeleteProfile(t *testing.T) {
	cfg := func(body string) string {
		return "; " + defaultComment + "\n" + body
	}

	cases := []struct {
		name            string
		seedConfig      string
		profileToDelete string
		wantSections    []string
		wantDefaultKeys map[string]string
	}{
		{
			name: "delete one of two profiles",
			seedConfig: cfg(`[DEFAULT]
[first]
host = https://first.cloud.databricks.com
[second]
host = https://second.cloud.databricks.com
`),
			profileToDelete: "first",
			wantSections:    []string{"DEFAULT", "second"},
		},
		{
			name: "delete last non-default profile",
			seedConfig: cfg(`[DEFAULT]
host = https://default.cloud.databricks.com
[only]
host = https://only.cloud.databricks.com
`),
			profileToDelete: "only",
			wantSections:    []string{"DEFAULT"},
			wantDefaultKeys: map[string]string{"host": "https://default.cloud.databricks.com"},
		},
		{
			name: "delete profile with multiple keys",
			seedConfig: cfg(`[DEFAULT]
[simple]
host = https://simple.cloud.databricks.com
[my-unified]
host = https://unified.cloud.databricks.com
account_id = def456
experimental_is_unified_host = true
`),
			profileToDelete: "my-unified",
			wantSections:    []string{"DEFAULT", "simple"},
		},
		{
			name: "delete default clears its keys and restores comment",
			seedConfig: cfg(`[DEFAULT]
host = https://default.cloud.databricks.com
[only]
host = https://only.cloud.databricks.com
`),
			profileToDelete: "DEFAULT",
			wantSections:    []string{"DEFAULT", "only"},
			wantDefaultKeys: map[string]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			path := filepath.Join(t.TempDir(), ".databrickscfg")
			require.NoError(t, os.WriteFile(path, []byte(tc.seedConfig), fileMode))

			err := DeleteProfile(ctx, tc.profileToDelete, path)
			require.NoError(t, err)

			file, err := config.LoadFile(path)
			require.NoError(t, err)

			var sectionNames []string
			for _, s := range file.Sections() {
				sectionNames = append(sectionNames, s.Name())
			}
			assert.Equal(t, tc.wantSections, sectionNames)

			defaultSection := file.Section(ini.DefaultSection)
			assert.Contains(t, defaultSection.Comment, defaultComment)
			if tc.wantDefaultKeys != nil {
				assert.Equal(t, tc.wantDefaultKeys, defaultSection.KeysHash())
			}
		})
	}
}

func TestClearDefaultProfile(t *testing.T) {
	cases := []struct {
		name        string
		initial     string
		profileName string
		wantDefault string
	}{
		{
			name:        "clears matching default",
			initial:     "[__settings__]\ndefault_profile = my-workspace\n\n[my-workspace]\nhost = https://abc\n\n[other]\nhost = https://def\n",
			profileName: "my-workspace",
			wantDefault: "",
		},
		{
			name:        "no-op when default differs",
			initial:     "[__settings__]\ndefault_profile = other\n\n[my-workspace]\nhost = https://abc\n\n[other]\nhost = https://def\n",
			profileName: "my-workspace",
			wantDefault: "other",
		},
		{
			name:        "no-op when no settings section",
			initial:     "[my-workspace]\nhost = https://abc\n",
			profileName: "my-workspace",
			wantDefault: "",
		},
		{
			name:        "no-op when no file",
			initial:     "",
			profileName: "my-workspace",
			wantDefault: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			path := filepath.Join(t.TempDir(), "databrickscfg")
			if tc.initial != "" {
				require.NoError(t, os.WriteFile(path, []byte(tc.initial), 0o600))
			}

			err := ClearDefaultProfile(ctx, tc.profileName, path)
			require.NoError(t, err)

			got, err := GetConfiguredDefaultProfile(ctx, path)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDefault, got)
		})
	}
}

func TestDeleteProfile_NotFound(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(""), fileMode))

	err := DeleteProfile(ctx, "not-found", path)
	require.Error(t, err)
	assert.ErrorContains(t, err, `profile "not-found" not found`)
}

func TestGetConfiguredAuthStorage(t *testing.T) {
	cases := []struct {
		name     string
		contents string
		want     string
	}{
		{
			name:     "missing settings section returns empty",
			contents: "[my-ws]\nhost = https://example.cloud.databricks.com\n",
			want:     "",
		},
		{
			name:     "settings without auth_storage returns empty",
			contents: "[__settings__]\ndefault_profile = my-ws\n",
			want:     "",
		},
		{
			name:     "explicit secure value",
			contents: "[__settings__]\nauth_storage = secure\n",
			want:     "secure",
		},
		{
			name:     "explicit plaintext value",
			contents: "[__settings__]\nauth_storage = plaintext\n",
			want:     "plaintext",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".databrickscfg")
			require.NoError(t, os.WriteFile(path, []byte(tc.contents), 0o600))

			got, err := GetConfiguredAuthStorage(t.Context(), path)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetConfiguredAuthStorage_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist")
	got, err := GetConfiguredAuthStorage(t.Context(), path)
	require.NoError(t, err)
	assert.Equal(t, "", got)
}
