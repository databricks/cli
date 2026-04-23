package deployment

import (
	"testing"

	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBindable(t *testing.T) {
	u := setupUcmFixture(t)

	cases := []struct {
		key     string
		want    phases.ImportKind
		wantErr string
	}{
		{"my_catalog", phases.ImportCatalog, ""},
		{"my_schema", phases.ImportSchema, ""},
		{"my_sc", phases.ImportStorageCredential, ""},
		{"my_loc", phases.ImportExternalLocation, ""},
		{"my_vol", phases.ImportVolume, ""},
		{"my_conn", phases.ImportConnection, ""},
		{"grant_a", "", "grants are not bindable"},
		{"does_not_exist", "", "no bindable resource"},
	}

	for _, c := range cases {
		got, err := resolveBindable(u, c.key)
		if c.wantErr != "" {
			require.Error(t, err, c.key)
			assert.Contains(t, err.Error(), c.wantErr, c.key)
			continue
		}
		require.NoError(t, err, c.key)
		assert.Equal(t, c.want, got, c.key)
	}
}

// TestBind_RejectsKindMismatch walks each bindable kind with a deliberately
// wrong-shape UC_NAME and asserts validateBindName surfaces a clear error
// up front rather than leaving the mismatch to the state writer.
func TestBind_RejectsKindMismatch(t *testing.T) {
	cases := []struct {
		name    string
		kind    phases.ImportKind
		ucName  string
		wantErr string
	}{
		{"catalog_with_dots", phases.ImportCatalog, "team_alpha.bronze", "must be a single identifier"},
		{"schema_without_dot", phases.ImportSchema, "bronze", "must be `<catalog>.<schema>`"},
		{"schema_too_many_dots", phases.ImportSchema, "team_alpha.bronze.landing", "must be `<catalog>.<schema>`"},
		{"volume_too_few_dots", phases.ImportVolume, "team_alpha.bronze", "must be `<catalog>.<schema>.<volume>`"},
		{"storage_credential_with_dot", phases.ImportStorageCredential, "some.cred", "must be a single identifier"},
		{"external_location_with_dot", phases.ImportExternalLocation, "a.b", "must be a single identifier"},
		{"connection_with_dot", phases.ImportConnection, "conn.x", "must be a single identifier"},
		{"empty_name", phases.ImportCatalog, "", "empty UC_NAME"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateBindName(c.kind, c.ucName)
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.wantErr)
		})
	}
}

// TestBind_AcceptsWellShapedNames locks in the happy-path shape for each kind
// so a future refactor of validateBindName doesn't silently tighten the spec.
func TestBind_AcceptsWellShapedNames(t *testing.T) {
	cases := []struct {
		name   string
		kind   phases.ImportKind
		ucName string
	}{
		{"catalog", phases.ImportCatalog, "team_alpha"},
		{"schema", phases.ImportSchema, "team_alpha.bronze"},
		{"volume", phases.ImportVolume, "team_alpha.bronze.landing"},
		{"storage_credential", phases.ImportStorageCredential, "sc1"},
		{"external_location", phases.ImportExternalLocation, "loc1"},
		{"connection", phases.ImportConnection, "conn1"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.NoError(t, validateBindName(c.kind, c.ucName))
		})
	}
}
