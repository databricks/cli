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
		name          string
		defaultName   string
		includeExtras bool
		wantNames     []string
		wantDefault   string
		wantExtras    []profilePickerResult
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
			name:          "extras appended after profiles",
			includeExtras: true,
			wantNames:     []string{"alpha", "bravo", "charlie", profilePickerCreateNewLabel, profilePickerEnterHostLabel},
			wantExtras:    []profilePickerResult{profilePickerCreateNew, profilePickerEnterHost},
		},
		{
			name:          "default first, then extras at the bottom",
			defaultName:   "charlie",
			includeExtras: true,
			wantNames:     []string{"charlie", "alpha", "bravo", profilePickerCreateNewLabel, profilePickerEnterHostLabel},
			wantDefault:   "charlie",
			wantExtras:    []profilePickerResult{profilePickerCreateNew, profilePickerEnterHost},
		},
		{
			name:        "default not in profiles is ignored",
			defaultName: "missing",
			wantNames:   []string{"alpha", "bravo", "charlie"},
			wantDefault: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := buildPickerItems(profiles, tc.defaultName, tc.includeExtras)

			gotNames := make([]string, len(items))
			gotDefault := ""
			var gotExtras []profilePickerResult
			for i, it := range items {
				gotNames[i] = it.Name
				if it.IsDefault {
					assert.Empty(t, gotDefault)
					gotDefault = it.Name
				}
				if it.IsExtra {
					gotExtras = append(gotExtras, it.Extra)
				}
			}
			assert.Equal(t, tc.wantNames, gotNames)
			assert.Equal(t, tc.wantDefault, gotDefault)
			assert.Equal(t, tc.wantExtras, gotExtras)
		})
	}
}
