package environments

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentVersion(t *testing.T) {
	cases := []struct {
		name string
		env  jobs.JobEnvironment
		want string
	}{
		{"nil spec", jobs.JobEnvironment{}, ""},
		{"environment_version", jobs.JobEnvironment{Spec: &compute.Environment{EnvironmentVersion: "3"}}, "3"},
		// client is the deprecated predecessor of environment_version; some jobs
		// still pin via it, so it must be read when environment_version is empty.
		{"client fallback", jobs.JobEnvironment{Spec: &compute.Environment{Client: "2"}}, "2"},
		// environment_version wins when both are present.
		{"environment_version wins", jobs.JobEnvironment{Spec: &compute.Environment{EnvironmentVersion: "4", Client: "2"}}, "4"},
		// base_environment is a path/ID, not a version, and is ignored.
		{"base_environment ignored", jobs.JobEnvironment{Spec: &compute.Environment{BaseEnvironment: "/Workspace/env.yaml"}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, environmentVersion(tc.env))
		})
	}
}
