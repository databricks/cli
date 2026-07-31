package phases

import (
	"net/url"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestResourceURL(t *testing.T) {
	baseURL := url.URL{Scheme: "https", Host: "example.databricks.test"}

	job := &resources.Job{
		BaseResource: resources.BaseResource{ID: "123"},
		JobSettings:  jobs.JobSettings{Name: "foo"},
	}
	job.InitializeURL(baseURL)
	byKey := map[string]config.ConfigResource{"resources.jobs.foo": job}

	tests := []struct {
		name   string
		action deployplan.Action
		want   string
	}{
		{
			name:   "live resource resolves via full plan key",
			action: deployplan.Action{ResourceKey: "resources.jobs.foo", ActionType: deployplan.Create},
			want:   "https://example.databricks.test/jobs/123",
		},
		{
			name:   "unknown key yields no URL",
			action: deployplan.Action{ResourceKey: "resources.jobs.missing", ActionType: deployplan.Create},
			want:   "",
		},
		{
			name:   "child node (permissions) yields no URL",
			action: deployplan.Action{ResourceKey: "resources.jobs.foo.permissions", ActionType: deployplan.Update},
			want:   "",
		},
		{
			name:   "delete builds URL from captured ID",
			action: deployplan.Action{ResourceKey: "resources.jobs.foo", ActionType: deployplan.Delete, ID: "456"},
			want:   "https://example.databricks.test/jobs/456",
		},
		{
			name:   "delete with empty ID yields no URL (no bogus /jobs/ link)",
			action: deployplan.Action{ResourceKey: "resources.jobs.foo", ActionType: deployplan.Delete, ID: ""},
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resourceURL(byKey, baseURL, tc.action))
		})
	}
}
