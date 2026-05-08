package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSelectItems(t *testing.T) {
	profiles := Profiles{
		{Name: "alpha", Host: "https://alpha.cloud.databricks.example"},
		{Name: "bravo", Host: "https://bravo.cloud.databricks.example"},
		{Name: "charlie", Host: "https://charlie.cloud.databricks.example"},
	}

	cases := []struct {
		name        string
		defaultName string
		wantOrder   []string
		wantDefault string
	}{
		{
			name:        "no default preserves order",
			defaultName: "",
			wantOrder:   []string{"alpha", "bravo", "charlie"},
			wantDefault: "",
		},
		{
			name:        "default in the middle moves to the top",
			defaultName: "bravo",
			wantOrder:   []string{"bravo", "alpha", "charlie"},
			wantDefault: "bravo",
		},
		{
			name:        "default already first stays first",
			defaultName: "alpha",
			wantOrder:   []string{"alpha", "bravo", "charlie"},
			wantDefault: "alpha",
		},
		{
			name:        "default not in profiles is ignored",
			defaultName: "missing",
			wantOrder:   []string{"alpha", "bravo", "charlie"},
			wantDefault: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := buildSelectItems(profiles, tc.defaultName)
			gotOrder := make([]string, len(items))
			gotDefault := ""
			for i, it := range items {
				gotOrder[i] = it.Name
				if it.IsDefault {
					assert.Empty(t, gotDefault, "more than one item flagged as default")
					gotDefault = it.Name
				}
			}
			assert.Equal(t, tc.wantOrder, gotOrder)
			assert.Equal(t, tc.wantDefault, gotDefault)
		})
	}
}

func TestBuildSelectItems_PaddedName(t *testing.T) {
	profiles := Profiles{
		{Name: "a"},
		{Name: "looooong"},
		{Name: "med"},
	}
	items := buildSelectItems(profiles, "")
	for _, it := range items {
		assert.Len(t, it.PaddedName, len("looooong"))
	}
}
