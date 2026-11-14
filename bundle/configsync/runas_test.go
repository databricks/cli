package configsync

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

const testRunAsPath = "resources.jobs.my_job.run_as"

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
		{"whole run_as backend default", testRunAsPath, backendDefault, false},
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
