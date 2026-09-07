package dresources

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

// assertFieldsCovered asserts that all fields in sdkType (except those in skip)
// are present as direct fields in remoteType, and that skipped fields are indeed absent.
func assertFieldsCovered(t *testing.T, sdkType, remoteType reflect.Type, skip map[string]bool) {
	t.Helper()
	remoteFields := map[string]bool{}
	for f := range remoteType.Fields() {
		if !f.Anonymous {
			remoteFields[f.Name] = true
		}
	}

	for field := range sdkType.Fields() {
		if skip[field.Name] {
			assert.NotContains(t, remoteFields, field.Name, "field %s is in skip list but present in %s; remove it from skip", field.Name, remoteType.Name())
			continue
		}
		assert.Contains(t, remoteFields, field.Name, "field %s from %s is missing in %s", field.Name, sdkType.Name(), remoteType.Name())
	}
}

func TestCollectUpdatePathsWithPrefix(t *testing.T) {
	upd := func() *deployplan.ChangeDesc { return &deployplan.ChangeDesc{Action: deployplan.Update} }
	skip := func() *deployplan.ChangeDesc { return &deployplan.ChangeDesc{Action: deployplan.Skip} }

	tests := []struct {
		name        string
		changes     Changes
		oneofGroups map[string]string
		want        []string
	}{
		{
			name:    "drops parent when a child is also updated",
			changes: Changes{"attributes": upd(), "attributes.createdb": upd()},
			want:    []string{"spec.attributes.createdb"},
		},
		{
			name:    "keeps parent when its only child is not updated",
			changes: Changes{"attributes": upd(), "attributes.createdb": skip()},
			want:    []string{"spec.attributes"},
		},
		{
			name:    "sorts multiple leaf paths",
			changes: Changes{"membership_roles": upd(), "attributes.createdb": upd()},
			want:    []string{"spec.attributes.createdb", "spec.membership_roles"},
		},
		{
			name:    "ignores non-update actions",
			changes: Changes{"parent": skip(), "role_id": skip(), "attributes.createdb": upd()},
			want:    []string{"spec.attributes.createdb"},
		},
		{
			name:    "no updates yields no paths",
			changes: Changes{"parent": skip()},
			want:    nil,
		},
		{
			name:        "renames a oneof member to its group",
			changes:     Changes{"ttl": upd()},
			oneofGroups: map[string]string{"ttl": "expiration"},
			want:        []string{"spec.expiration"},
		},
		{
			name:        "collapses two members of one oneof",
			changes:     Changes{"ttl": upd(), "no_expiry": upd()},
			oneofGroups: map[string]string{"ttl": "expiration", "no_expiry": "expiration"},
			want:        []string{"spec.expiration"},
		},
		{
			name:    "masks a map as a whole",
			changes: Changes{"settings.pg_settings['work_mem']": upd(), "settings.pg_settings['jit']": upd()},
			want:    []string{"spec.settings.pg_settings"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, collectUpdatePathsWithPrefix(tc.changes, "spec.", tc.oneofGroups))
		})
	}
}
