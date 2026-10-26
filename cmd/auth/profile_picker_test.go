package auth

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
)

func TestBuildPickerItems(t *testing.T) {
	profiles := profile.Profiles{
		{Name: "alpha", Host: "https://alpha.cloud.databricks.example"},
		{Name: "bravo", Host: "https://bravo.cloud.databricks.example"},
		{Name: "charlie", Host: "https://charlie.cloud.databricks.example"},
	}

	cases := []struct {
		name             string
		defaultName      string
		includeCreateNew bool
		includeEnterHost bool
		wantNames        []string
		wantDefault      string
		wantExtras       []profilePickerResult
	}{
		{
			name:        "no default no extras",
			wantNames:   []string{"alpha", "bravo", "charlie"},
			wantDefault: "",
		},
		{
			name:        "default moves to top",
			defaultName: "bravo",
			wantNames:   []string{"bravo", "alpha", "charlie"},
			wantDefault: "bravo",
		},
		{
			name:             "extras appended after profiles",
			includeCreateNew: true,
			includeEnterHost: true,
			wantNames:        []string{"alpha", "bravo", "charlie", profilePickerCreateNewLabel, profilePickerEnterHostLabel},
			wantExtras:       []profilePickerResult{profilePickerCreateNew, profilePickerEnterHost},
		},
		{
			name:             "default first, then extras at the bottom",
			defaultName:      "charlie",
			includeCreateNew: true,
			includeEnterHost: true,
			wantNames:        []string{"charlie", "alpha", "bravo", profilePickerCreateNewLabel, profilePickerEnterHostLabel},
			wantDefault:      "charlie",
			wantExtras:       []profilePickerResult{profilePickerCreateNew, profilePickerEnterHost},
		},
		{
			name:             "only enter host",
			includeEnterHost: true,
			wantNames:        []string{"alpha", "bravo", "charlie", profilePickerEnterHostLabel},
			wantExtras:       []profilePickerResult{profilePickerEnterHost},
		},
		{
			name:             "only create new",
			includeCreateNew: true,
			wantNames:        []string{"alpha", "bravo", "charlie", profilePickerCreateNewLabel},
			wantExtras:       []profilePickerResult{profilePickerCreateNew},
		},
		{
			name:        "default not in profiles is ignored",
			defaultName: "missing",
			wantNames:   []string{"alpha", "bravo", "charlie"},
			wantDefault: "",
		},
	}

	t.Run("empty profiles with extras shows only extras", func(t *testing.T) {
		items := buildPickerItems(profile.Profiles{}, "", true, true)
		assert.Equal(t, []string{profilePickerCreateNewLabel, profilePickerEnterHostLabel}, namesOf(items))
	})

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := buildPickerItems(profiles, tc.defaultName, tc.includeCreateNew, tc.includeEnterHost)

			gotDefault := ""
			var gotExtras []profilePickerResult
			for _, it := range items {
				if it.IsDefault {
					assert.Empty(t, gotDefault)
					gotDefault = it.Name
				}
				if it.IsExtra {
					gotExtras = append(gotExtras, it.Extra)
				}
			}
			assert.Equal(t, tc.wantNames, namesOf(items))
			assert.Equal(t, tc.wantDefault, gotDefault)
			assert.Equal(t, tc.wantExtras, gotExtras)
		})
	}
}

func namesOf(items []pickerItem) []string {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.Name
	}
	return names
}
