package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

const (
	testRunAsPath = "resources.jobs.my_job.run_as"
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
	owner := "owner@example.com"

	tests := []struct {
		name string
		path string
		cd   *deployplan.ChangeDesc
		want bool
	}{
		{"set remotely to another principal", testRunAsPath, &deployplan.ChangeDesc{Remote: sp}, false},
		{"set remotely as a pointer", testRunAsPath, &deployplan.ChangeDesc{Remote: &sp}, false},
		{"set remotely to the owner", testRunAsPath, &deployplan.ChangeDesc{Remote: jobs.JobRunAs{UserName: owner}}, true},
		{"set remotely to the owner as a service principal", testRunAsPath, &deployplan.ChangeDesc{Remote: jobs.JobRunAs{ServicePrincipalName: owner}}, true},
		{"removed from config, kept remotely", testRunAsPath, &deployplan.ChangeDesc{Old: user, Remote: user}, false},
		{"set in config and remotely", testRunAsPath, &deployplan.ChangeDesc{New: user, Remote: sp}, true},
		{"removed remotely", testRunAsPath, &deployplan.ChangeDesc{Old: user, New: user}, true},
		{"removed from config and remotely", testRunAsPath, &deployplan.ChangeDesc{Old: user}, true},
		{"run_as key", testRunAsPath + ".service_principal_name", &deployplan.ChangeDesc{Remote: testSPID}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, skipJobRunAs(tt.path, tt.cd, owner))
		})
	}

	// Without a known owner, no remote run_as is synced.
	assert.True(t, skipJobRunAs(testRunAsPath, &deployplan.ChangeDesc{Remote: sp}, ""))
}

func TestJobOwners(t *testing.T) {
	const jobKey = "resources.jobs.my_job"
	unsetRunAs := map[string]*deployplan.ChangeDesc{"run_as": {Remote: jobs.JobRunAs{UserName: "user@example.com"}}}
	job := &deployplan.PlanEntry{
		RemoteState: &dresources.JobRemote{CreatorUserName: "creator@example.com"},
		Changes:     unsetRunAs,
	}
	acl := []dresources.StatePermission{
		{Level: iam.PermissionLevelCanManage, UserName: "manager@example.com"},
		{Level: iam.PermissionLevelIsOwner, ServicePrincipalName: testSPID},
	}

	tests := []struct {
		name   string
		plan   map[string]*deployplan.PlanEntry
		remote map[string][]dresources.StatePermission
		want   map[string]string
	}{
		{
			name:   "owner from the unmanaged ACL",
			plan:   map[string]*deployplan.PlanEntry{jobKey: job},
			remote: map[string][]dresources.StatePermission{jobKey: acl},
			want:   map[string]string{jobKey: testSPID},
		},
		{
			name: "owner from the declared permissions",
			plan: map[string]*deployplan.PlanEntry{
				jobKey:                  job,
				jobKey + ".permissions": {RemoteState: &dresources.PermissionsState{EmbeddedSlice: acl}},
			},
			want: map[string]string{jobKey: testSPID},
		},
		{
			name: "owner from the undeployed permissions",
			plan: map[string]*deployplan.PlanEntry{
				jobKey:                  job,
				jobKey + ".permissions": {Action: deployplan.Create},
			},
			remote: map[string][]dresources.StatePermission{jobKey: acl},
			want:   map[string]string{jobKey: testSPID},
		},
		{
			name:   "creator when the ACL has no owner",
			plan:   map[string]*deployplan.PlanEntry{jobKey: job},
			remote: map[string][]dresources.StatePermission{jobKey: acl[:1]},
			want:   map[string]string{jobKey: "creator@example.com"},
		},
		{
			name: "unknown when the ACL was not read",
			plan: map[string]*deployplan.PlanEntry{jobKey: job},
			want: map[string]string{},
		},
		{
			name: "unknown when declared permissions can't be read",
			plan: map[string]*deployplan.PlanEntry{jobKey: job, jobKey + ".permissions": {}},
			want: map[string]string{},
		},
		{
			name: "run_as set in config",
			plan: map[string]*deployplan.PlanEntry{jobKey: {
				RemoteState: job.RemoteState,
				Changes:     map[string]*deployplan.ChangeDesc{"run_as": {New: jobs.JobRunAs{UserName: "user@example.com"}, Remote: jobs.JobRunAs{UserName: "user@example.com"}}},
			}},
			want: map[string]string{},
		},
		{
			name: "not a job",
			plan: map[string]*deployplan.PlanEntry{jobKey: {RemoteState: &dresources.PermissionsState{}, Changes: unsetRunAs}},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, jobOwners(t.Context(), &deployplan.Plan{Plan: tt.plan}, tt.remote))
		})
	}
}
