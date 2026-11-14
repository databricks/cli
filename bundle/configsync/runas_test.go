package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

const (
	testRunAsPath = "resources.jobs.my_job.run_as"
	testCreator   = "creator@example.com"
	testSPID      = "11111111-2222-3333-4444-555555555555"
)

func TestKeepSkippedRunAsChange(t *testing.T) {
	backendDefault := &deployplan.ChangeDesc{Action: deployplan.Skip, Reason: deployplan.ReasonBackendDefault}
	tests := []struct {
		name string
		path string
		cd   *deployplan.ChangeDesc
		want bool
	}{
		{"service_principal_name backend default", testRunAsPath + ".service_principal_name", backendDefault, true},
		{"user_name backend default", testRunAsPath + ".user_name", backendDefault, true},
		{"group_name backend default", testRunAsPath + ".group_name", backendDefault, true},
		{"whole run_as backend default", testRunAsPath, backendDefault, true},
		{"run_as key already matches the remote", testRunAsPath + ".user_name", &deployplan.ChangeDesc{Action: deployplan.Skip, Reason: deployplan.ReasonRemoteAlreadySet}, false},
		{"other job field", "resources.jobs.my_job.tasks[task_key='main'].new_cluster.enable_elastic_disk", backendDefault, false},
		{"pipeline run_as key", "resources.pipelines.my_pipeline.run_as.user_name", backendDefault, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, keepSkippedRunAsChange(tt.path, tt.cd))
		})
	}
}

func TestSkipJobRunAs(t *testing.T) {
	sp := jobs.JobRunAs{ServicePrincipalName: testSPID}
	user := jobs.JobRunAs{UserName: "user@example.com"}
	creator := jobs.JobRunAs{UserName: testCreator}
	group := jobs.JobRunAs{GroupName: "data-engineers"}

	tests := []struct {
		name    string
		path    string
		cd      *deployplan.ChangeDesc
		creator string
		want    bool
	}{
		{
			name:    "service principal set remotely",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Remote: sp},
			creator: testCreator,
			want:    false,
		},
		{
			name:    "group set remotely",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Remote: group},
			creator: testCreator,
			want:    false,
		},
		{
			name:    "user that is the creator",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Remote: creator},
			creator: testCreator,
			want:    true,
		},
		{
			name:    "service principal that is the creator",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Remote: sp},
			creator: testSPID,
			want:    true,
		},
		{
			name:    "unchanged since the last deploy",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Old: user, Remote: user},
			creator: testCreator,
			want:    true,
		},
		{
			name:    "changed since the last deploy",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Old: user, Remote: sp},
			creator: testCreator,
			want:    false,
		},
		{
			name:    "removed remotely",
			path:    testRunAsPath,
			cd:      &deployplan.ChangeDesc{Old: user, New: user},
			creator: testCreator,
			want:    true,
		},
		{
			name:    "run_as key",
			path:    testRunAsPath + ".service_principal_name",
			cd:      &deployplan.ChangeDesc{Remote: testSPID},
			creator: testSPID,
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remoteState := &dresources.JobRemote{CreatorUserName: tt.creator}
			assert.Equal(t, tt.want, skipJobRunAs(tt.path, tt.cd, remoteState))
		})
	}
}
