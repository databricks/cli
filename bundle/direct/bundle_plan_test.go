package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestDynPathToStructPath(t *testing.T) {
	tests := []struct {
		path     dyn.Path
		expected string
	}{
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Key("bar")),
			expected: "foo.bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar")),
			expected: "foo[1].bar",
		},
		{
			path:     dyn.NewPath(dyn.Key("configuration"), dyn.Key("europris.swipe.egress_streaming_schema")),
			expected: "configuration['europris.swipe.egress_streaming_schema']",
		},
		{
			path:     dyn.NewPath(dyn.Key("tags"), dyn.Key("it's.here")),
			expected: "tags['it''s.here']",
		},
	}

	for _, tc := range tests {
		node := dynPathToStructPath(tc.path)
		assert.Equal(t, tc.expected, node.String())
	}
}

func TestValidatePlanAgainstState(t *testing.T) {
	stateDB := func(lineage string, serial int) *dstate.DeploymentState {
		return &dstate.DeploymentState{
			Path: "state.json",
			Data: dstate.Database{Header: dstate.Header{Lineage: lineage, Serial: serial}},
		}
	}

	tests := []struct {
		name       string
		plan       *deployplan.Plan
		stateDB    *dstate.DeploymentState
		dmsEnabled bool
		wantErr    string
	}{
		{
			name:    "no lineage skips validation",
			plan:    &deployplan.Plan{},
			stateDB: stateDB("abc", 5),
		},
		{
			name:    "lineage mismatch is rejected",
			plan:    &deployplan.Plan{Lineage: "abc", Serial: 5},
			stateDB: stateDB("xyz", 5),
			wantErr: "plan lineage \"abc\" does not match state lineage \"xyz\"",
		},
		{
			name:    "matching serial passes",
			plan:    &deployplan.Plan{Lineage: "abc", Serial: 5},
			stateDB: stateDB("abc", 5),
		},
		{
			name:    "serial mismatch is rejected when DMS is off",
			plan:    &deployplan.Plan{Lineage: "abc", Serial: 4},
			stateDB: stateDB("abc", 5),
			wantErr: "plan serial 4 does not match state serial 5",
		},
		{
			name:       "serial is ignored when DMS is on",
			plan:       &deployplan.Plan{Lineage: "abc", VersionId: "3"},
			stateDB:    stateDB("abc", 5),
			dmsEnabled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePlanAgainstState(tc.stateDB, tc.plan, tc.dmsEnabled)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
