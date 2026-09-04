package dresources

import (
	"reflect"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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

	suspension := map[string]string{
		"default_endpoint_settings.suspend_timeout_duration": "default_endpoint_settings.suspension",
	}

	tests := []struct {
		name        string
		changes     Changes
		oneofGroups map[string]string
		spec        any
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
		{
			name:    "expands a whole new message to the fields the body populates",
			changes: Changes{"default_endpoint_settings": upd()},
			spec: &postgres.ProjectSpec{
				DefaultEndpointSettings: &postgres.ProjectDefaultEndpointSettings{AutoscalingLimitMinCu: 0.5},
			},
			want: []string{"spec.default_endpoint_settings.autoscaling_limit_min_cu"},
		},
		{
			name:    "renames a oneof member the expansion reaches",
			changes: Changes{"default_endpoint_settings": upd()},
			spec: &postgres.ProjectSpec{
				DefaultEndpointSettings: &postgres.ProjectDefaultEndpointSettings{
					AutoscalingLimitMinCu:  0.5,
					SuspendTimeoutDuration: duration.New(300 * time.Second),
				},
			},
			oneofGroups: suspension,
			want:        []string{"spec.default_endpoint_settings.autoscaling_limit_min_cu", "spec.default_endpoint_settings.suspension"},
		},
		{
			name:    "keeps the message when the body leaves nothing under it unset",
			changes: Changes{"settings": upd()},
			spec: &postgres.EndpointSpec{
				Settings: &postgres.EndpointSettings{PgSettings: map[string]string{"work_mem": "4MB"}},
			},
			want: []string{"spec.settings"},
		},
		{
			name:    "expands a partly populated message and keeps its map whole",
			changes: Changes{"group": upd()},
			spec: &postgres.EndpointSpec{
				Group: &postgres.EndpointGroupSpec{Min: 1, Max: 1},
			},
			want: []string{"spec.group.max", "spec.group.min"},
		},
		{
			name:    "does not expand a repeated field",
			changes: Changes{"custom_tags": upd()},
			spec: &postgres.ProjectSpec{
				CustomTags: []postgres.ProjectCustomTag{{Key: "release_id", Value: "1"}},
			},
			want: []string{"spec.custom_tags"},
		},
		{
			name:        "does not expand a timestamp wrapper",
			changes:     Changes{"expire_time": upd()},
			spec:        &postgres.BranchSpec{ExpireTime: sdktime.New(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))},
			oneofGroups: map[string]string{"expire_time": "expiration"},
			want:        []string{"spec.expiration"},
		},
		{
			name:    "keeps the message when the body populates nothing under it",
			changes: Changes{"default_endpoint_settings": upd()},
			spec:    &postgres.ProjectSpec{DefaultEndpointSettings: &postgres.ProjectDefaultEndpointSettings{}},
			want:    []string{"spec.default_endpoint_settings"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, collectUpdatePathsWithPrefix(tc.changes, "spec.", tc.oneofGroups, tc.spec))
		})
	}
}
