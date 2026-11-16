package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	tests := []struct {
		name     string
		path     string
		cd       *deployplan.ChangeDesc
		selected bool
		want     bool
	}{
		{
			name:     "set remotely on a selected job",
			path:     testRunAsPath,
			cd:       &deployplan.ChangeDesc{Remote: sp},
			selected: true,
			want:     false,
		},
		{
			name: "set remotely on a job that is not selected",
			path: testRunAsPath,
			cd:   &deployplan.ChangeDesc{Remote: sp},
			want: true,
		},
		{
			name:     "removed from config, kept remotely, selected",
			path:     testRunAsPath,
			cd:       &deployplan.ChangeDesc{Old: user, Remote: user},
			selected: true,
			want:     false,
		},
		{
			name:     "set in config and remotely",
			path:     testRunAsPath,
			cd:       &deployplan.ChangeDesc{New: user, Remote: sp},
			selected: true,
			want:     true,
		},
		{
			name:     "removed remotely",
			path:     testRunAsPath,
			cd:       &deployplan.ChangeDesc{Old: user, New: user},
			selected: true,
			want:     true,
		},
		{
			name:     "removed from config and remotely",
			path:     testRunAsPath,
			cd:       &deployplan.ChangeDesc{Old: user},
			selected: true,
			want:     true,
		},
		{
			name:     "run_as key",
			path:     testRunAsPath + ".service_principal_name",
			cd:       &deployplan.ChangeDesc{Remote: testSPID},
			selected: false,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, skipJobRunAs(tt.path, tt.cd, tt.selected))
		})
	}
}

func TestParseRunAsSelection(t *testing.T) {
	selection, err := ParseRunAsSelection([]string{"jobs:123", "jobs:456"})
	require.NoError(t, err)
	deployed := map[string]string{
		"jobs:123":      "resources.jobs.first",
		"jobs:789":      "resources.jobs.other",
		"pipelines:456": "resources.pipelines.pipeline",
	}
	assert.Equal(t, map[string]bool{"resources.jobs.first": true}, selection.ResourceKeys(t.Context(), deployed))

	for _, selector := range []string{"123", "jobs:", "pipelines:123"} {
		_, err := ParseRunAsSelection([]string{selector})
		assert.ErrorContains(t, err, "invalid --sync-run-as-ids value", selector)
	}
}
