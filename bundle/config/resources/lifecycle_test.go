package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Resources that shadow BaseResource.Lifecycle must return their own settings.
func TestGetLifecycleDependsOn(t *testing.T) {
	deps := []DependsOn{{Resource: "foo"}}

	tests := []struct {
		name     string
		resource interface{ GetLifecycle() LifecycleConfig }
	}{
		{"base", &Job{Lifecycle: Lifecycle{DependsOn: deps}}},
		{"with_started", &App{Lifecycle: &LifecycleWithStarted{DependsOn: deps}}},
		{"job_run", &JobRun{Lifecycle: &JobRunLifecycle{DependsOn: deps}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, deps, tc.resource.GetLifecycle().GetDependsOn())
		})
	}
}

func TestJobRunGetLifecycleNil(t *testing.T) {
	assert.Empty(t, (&JobRun{}).GetLifecycle().GetDependsOn())
}
