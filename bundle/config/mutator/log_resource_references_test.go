package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config"
	cres "github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestConvertReferenceToMetric_Table(t *testing.T) {
	ctx := context.Background()
	cfg := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*cres.Job{
				"foo": {
					JobSettings: jobs.JobSettings{
						Tasks: []jobs.Task{
							{TaskKey: "alpha"},
						},
					},
				},
				"niljob": nil,
			},
			Apps: map[string]*cres.App{
				"app1": {
					Config: map[string]any{
						"k": "v",
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "basic job id",
			ref:  "resources.jobs.foo.id",
			want: "resref_jobs.id",
		},
		{
			name: "basic job id with non-ascii key",
			ref:  "resources.jobs.джоб.id",
			want: "resreferr_jobs",
		},
		{
			name: "invalid empty",
			ref:  "",
			want: "",
		},
		{
			name: "invalid variables",
			ref:  "variables.foo",
			want: "",
		},
		{
			name: "invalid short resources",
			ref:  "resources",
			want: "",
		},
		{
			name: "invalid short group",
			ref:  "resources.jobs",
			want: "",
		},
		{
			name: "mapkey censor on app config",
			ref:  "resources.apps.app1.config.foo",
			want: "resref_apps.config.*",
		},
		{
			name: "nil job pointer yields plain id",
			ref:  "resources.jobs.niljob.id",
			want: "resref_jobs.id",
		},
		{
			name: "err censor on missing job key",
			ref:  "resources.jobs.missing.id",
			want: "resreferr_jobs",
		},
		{
			name: "array index tasks key",
			ref:  "resources.jobs.foo.tasks[0].task_key",
			want: "resref_jobs.tasks.task_key",
		},
		{
			name: "array index task key",
			ref:  "resources.jobs.foo.task[0].task_key",
			want: "resreferr_jobs.task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertReferenceToMetric(ctx, cfg, tt.ref)
			assert.Equal(t, tt.want, got)
		})
	}
}
