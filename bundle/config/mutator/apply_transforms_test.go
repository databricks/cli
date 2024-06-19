package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		job    *resources.Job
		want   string
	}{
		{
			name:   "add prefix to job",
			prefix: "prefix-",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
				},
			},
			want: "prefix-job1",
		},
		{
			name:   "add empty prefix to job",
			prefix: "",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
				},
			},
			want: "job1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": tt.job,
						},
					},
					Transform: config.Transforms{
						Prefix: tt.prefix,
					},
				},
			}

			mutator := mutator.ApplyTransforms()
			diag := mutator.Apply(context.Background(), b)

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			require.Equal(t, tt.want, b.Config.Resources.Jobs["job1"].Name)
		})
	}
}

func TestTransformTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		job  *resources.Job
		want map[string]string
	}{
		{
			name: "add tags to job",
			tags: map[string]string{"env": "dev"},
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
					Tags: nil,
				},
			},
			want: map[string]string{"env": "dev"},
		},
		{
			name: "merge tags with existing job tags",
			tags: map[string]string{"env": "dev"},
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
					Tags: map[string]string{"team": "data"},
				},
			},
			want: map[string]string{"env": "dev", "team": "data"},
		},
		{
			name: "don't override existing job tags",
			tags: map[string]string{"env": "dev"},
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name: "job1",
					Tags: map[string]string{"env": "prod"},
				},
			},
			want: map[string]string{"env": "prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": tt.job,
						},
					},
					Transform: config.Transforms{
						Tags: &tt.tags,
					},
				},
			}

			mutator := mutator.ApplyTransforms()
			diag := mutator.Apply(context.Background(), b)

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			tags := b.Config.Resources.Jobs["job1"].Tags
			require.Equal(t, tt.want, tags)
		})
	}
}

func TestTransformJobsMaxConcurrentRuns(t *testing.T) {
	tests := []struct {
		name    string
		job     *resources.Job
		setting int
		want    int
	}{
		{
			name: "set max concurrent runs",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name:              "job1",
					MaxConcurrentRuns: 0,
				},
			},
			setting: 5,
			want:    5,
		},
		{
			name: "do not override existing max concurrent runs",
			job: &resources.Job{
				JobSettings: &jobs.JobSettings{
					Name:              "job1",
					MaxConcurrentRuns: 3,
				},
			},
			setting: 5,
			want:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Jobs: map[string]*resources.Job{
							"job1": tt.job,
						},
					},
					Transform: config.Transforms{
						DefaultJobsMaxConcurrentRuns: tt.setting,
					},
				},
			}

			mutator := mutator.ApplyTransforms()
			diag := mutator.Apply(context.Background(), b)

			if diag.HasError() {
				t.Fatalf("unexpected error: %v", diag)
			}

			require.Equal(t, tt.want, b.Config.Resources.Jobs["job1"].MaxConcurrentRuns)
		})
	}
}

func TestTransformValidation(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Transform: config.Transforms{
				DefaultTriggerPauseStatus: "invalid",
			},
		},
	}

	mutator := mutator.ApplyTransforms()
	diag := mutator.Apply(context.Background(), b)
	assert.Equal(t, "Invalid value for default_trigger_pause_status, should be PAUSED or UNPAUSED", diag[0].Summary)
}
