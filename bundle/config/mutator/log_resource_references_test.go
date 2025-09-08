package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config"
	cres "github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestConvertReferenceToMetric_Table(t *testing.T) {
	ctx := context.Background()
	cfg := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*cres.Job{
				"foo":    {},
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
		{name: "basic job id", ref: "resources.jobs.foo.id", want: "resref__jobs__id"},
		{name: "invalid empty", ref: "", want: ""},
		{name: "invalid variables", ref: "variables.foo", want: ""},
		{name: "invalid short resources", ref: "resources", want: ""},
		{name: "invalid short group", ref: "resources.jobs", want: ""},
		{name: "mapkey censor on app config", ref: "resources.apps.app1.config.foo", want: "resref__apps__config__mapkey"},
		{name: "nil job pointer yields plain id", ref: "resources.jobs.niljob.id", want: "resref__jobs__id"},
		{name: "err censor on missing job key", ref: "resources.jobs.missing.id", want: "resreferr__jobs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertReferenceToMetric(ctx, cfg, tt.ref)
			assert.Equal(t, tt.want, got)
		})
	}
}
