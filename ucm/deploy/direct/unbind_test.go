package direct_test

import (
	"testing"

	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedUnbindState returns a fresh state with one entry under every kind
// UnbindResource knows about. Tests mutate (or inspect) only the slice they
// care about so mutual independence is preserved.
func seedUnbindState() *direct.State {
	s := direct.NewState()
	s.Catalogs["main"] = &direct.CatalogState{Name: "main"}
	s.Schemas["raw"] = &direct.SchemaState{Name: "raw", Catalog: "main"}
	s.StorageCredentials["sc"] = &direct.StorageCredentialState{Name: "sc"}
	s.ExternalLocations["loc"] = &direct.ExternalLocationState{Name: "loc"}
	s.Volumes["vol"] = &direct.VolumeState{Name: "vol", CatalogName: "main", SchemaName: "raw"}
	s.Connections["conn"] = &direct.ConnectionState{Name: "conn"}
	return s
}

func TestUnbindResource_PerKindHappyPath(t *testing.T) {
	cases := []struct {
		kind string
		key  string
		// lookup returns (present, _) after unbind so the test can assert the
		// entry is gone without duplicating the switch per kind in the test body.
		lookup func(s *direct.State, key string) bool
	}{
		{"catalog", "main", func(s *direct.State, k string) bool { _, ok := s.Catalogs[k]; return ok }},
		{"schema", "raw", func(s *direct.State, k string) bool { _, ok := s.Schemas[k]; return ok }},
		{"storage_credential", "sc", func(s *direct.State, k string) bool { _, ok := s.StorageCredentials[k]; return ok }},
		{"external_location", "loc", func(s *direct.State, k string) bool { _, ok := s.ExternalLocations[k]; return ok }},
		{"volume", "vol", func(s *direct.State, k string) bool { _, ok := s.Volumes[k]; return ok }},
		{"connection", "conn", func(s *direct.State, k string) bool { _, ok := s.Connections[k]; return ok }},
	}

	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			state := seedUnbindState()
			require.True(t, c.lookup(state, c.key), "precondition: %s[%q] must be seeded", c.kind, c.key)

			require.NoError(t, direct.UnbindResource(t.Context(), state, c.kind, c.key))
			assert.False(t, c.lookup(state, c.key), "%s[%q] must be removed after unbind", c.kind, c.key)
		})
	}
}

func TestUnbindResource_NotCurrentlyBoundErrors(t *testing.T) {
	cases := []struct{ kind, key string }{
		{"catalog", "ghost"},
		{"schema", "ghost"},
		{"storage_credential", "ghost"},
		{"external_location", "ghost"},
		{"volume", "ghost"},
		{"connection", "ghost"},
	}

	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			state := direct.NewState()

			err := direct.UnbindResource(t.Context(), state, c.kind, c.key)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no bound")
			assert.Contains(t, err.Error(), c.key)
		})
	}
}

func TestUnbindResource_UnsupportedKindErrors(t *testing.T) {
	state := direct.NewState()

	err := direct.UnbindResource(t.Context(), state, "table", "foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}
