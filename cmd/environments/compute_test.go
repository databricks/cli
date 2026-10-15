package environments

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentVersion(t *testing.T) {
	cases := []struct {
		name string
		env  jobs.JobEnvironment
		want string
	}{
		{"nil spec", jobs.JobEnvironment{}, ""},
		{"environment_version", jobs.JobEnvironment{Spec: &jobs.Environment{EnvironmentVersion: new("3")}}, "3"},
		// client is the deprecated predecessor of environment_version; some jobs
		// still pin via it, so it must be read when environment_version is empty.
		{"client fallback", jobs.JobEnvironment{Spec: &jobs.Environment{Client: new("2")}}, "2"},
		// environment_version wins when both are present.
		{"environment_version wins", jobs.JobEnvironment{Spec: &jobs.Environment{EnvironmentVersion: new("4"), Client: new("2")}}, "4"},
		// base_environment is a path/ID, not a version, and is ignored.
		{"base_environment ignored", jobs.JobEnvironment{Spec: &jobs.Environment{BaseEnvironment: new("/Workspace/env.yaml")}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, environmentVersion(tc.env))
		})
	}
}
