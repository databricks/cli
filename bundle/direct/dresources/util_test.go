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
	updTo := func(v any) *deployplan.ChangeDesc {
		return &deployplan.ChangeDesc{Action: deployplan.Update, New: v}
	}
	suspension := map[string]string{
		"default_endpoint_settings.suspend_timeout_duration": "default_endpoint_settings.suspension",
	}

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
		{
			name: "expands a whole new message to the fields the body populates",
			changes: Changes{"default_endpoint_settings": updTo(&postgres.ProjectDefaultEndpointSettings{
				AutoscalingLimitMinCu: 0.5,
			})},
			want: []string{"spec.default_endpoint_settings.autoscaling_limit_min_cu"},
		},
		{
			name: "renames a oneof member the expansion reaches",
			changes: Changes{"default_endpoint_settings": updTo(&postgres.ProjectDefaultEndpointSettings{
				SuspendTimeoutDuration: duration.New(300 * time.Second),
			})},
			oneofGroups: suspension,
			want:        []string{"spec.default_endpoint_settings.suspension"},
		},
		{
			name: "keeps a map inside an expanded message whole",
			changes: Changes{"settings": updTo(&postgres.EndpointSettings{
				PgSettings: map[string]string{"work_mem": "4MB"},
			})},
			want: []string{"spec.settings.pg_settings"},
		},
		{
			name:    "does not expand a repeated field",
			changes: Changes{"custom_tags": updTo([]postgres.ProjectCustomTag{{Key: "release_id", Value: "1"}})},
			want:    []string{"spec.custom_tags"},
		},
		{
			name:        "does not expand a timestamp wrapper",
			changes:     Changes{"expire_time": updTo(sdktime.New(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)))},
			oneofGroups: map[string]string{"expire_time": "expiration"},
			want:        []string{"spec.expiration"},
		},
		{
			name:    "keeps the message when the body populates nothing under it",
			changes: Changes{"default_endpoint_settings": updTo(&postgres.ProjectDefaultEndpointSettings{})},
			want:    []string{"spec.default_endpoint_settings"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, collectUpdatePathsWithPrefix(tc.changes, "spec.", tc.oneofGroups))
		})
	}
}
