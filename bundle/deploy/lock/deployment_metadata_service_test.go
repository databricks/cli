package lock

import (
	"testing"

	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
)

func TestGoalToVersionType(t *testing.T) {
	tests := []struct {
		goal    Goal
		want    sdkbundle.VersionType
		wantErr bool
	}{
		{goal: GoalDeploy, want: sdkbundle.VersionTypeVersionTypeDeploy},
		{goal: GoalDestroy, want: sdkbundle.VersionTypeVersionTypeDestroy},
		{goal: GoalBind, wantErr: true},
		{goal: GoalUnbind, wantErr: true},
		{goal: Goal("garbage"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(string(tt.goal), func(t *testing.T) {
			got, err := goalToVersionType(tt.goal)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNextVersionID(t *testing.T) {
	tests := []struct {
		name    string
		last    string
		want    string
		wantErr bool
	}{
		{name: "empty starts at 1", last: "", want: "1"},
		{name: "increments numeric", last: "1", want: "2"},
		{name: "increments larger numeric", last: "42", want: "43"},
		{name: "rejects non-numeric", last: "v1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nextVersionID(tt.last)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
