package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

type getResourceKeyTestCase struct {
	path string
	key  ResourceKey
	err  bool
}

func TestGetResourceKey(t *testing.T) {
	testCases := []getResourceKeyTestCase{
		{
			path: "resources.jobs.job_1",
			key: ResourceKey{
				Type: "jobs",
				Name: "job_1",
			},
		},
		{
			path: "resources.jobs.job_1.name",
			key: ResourceKey{
				Type: "jobs",
				Name: "job_1",
			},
		},
		{
			path: "resources.jobs",
			err:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			key, err := getResourceKey(structpath.MustParsePath(tc.path))
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.key, key)
			}
		})
	}
}

type resourceKeySetAddTestCase struct {
	name     string
	pattern  *structpath.PatternNode
	root     structvar.View
	expected []ResourceKey
}

func TestResourceKeySet_AddPattern(t *testing.T) {
	var cfg config.Root
	cfg.Resources.Jobs = map[string]*resources.Job{
		"job_1": {JobSettings: jobs.JobSettings{Name: "job_1"}},
		"job_2": {JobSettings: jobs.JobSettings{Name: "job_2"}},
	}
	root := cfg.View()

	testCases := []resourceKeySetAddTestCase{
		{
			name:    "one job pattern",
			pattern: structpath.MustParsePattern("resources.jobs.job_1"),
			root:    root,
			expected: []ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
			},
		},
		{
			name:    "all resources pattern",
			pattern: structpath.MustParsePattern("resources.*.*"),
			root:    root,
			expected: []ResourceKey{
				{
					Type: "jobs",
					Name: "job_1",
				},
				{
					Type: "jobs",
					Name: "job_2",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			set := NewResourceKeySet()

			err := set.AddPattern(tc.pattern, tc.root)

			assert.NoError(t, err)
			assert.ElementsMatch(t, tc.expected, set.ToArray())
		})
	}
}

func TestResourceKeySet_AddResourceKey(t *testing.T) {
	set := NewResourceKeySet()

	set.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_1"})
	set.AddResourceKey(ResourceKey{Type: "pipelines", Name: "pipeline_1"})

	assert.ElementsMatch(t, []string{"jobs", "pipelines"}, set.Types())
	assert.ElementsMatch(t, []string{"job_1"}, set.Names("jobs"))
	assert.ElementsMatch(t, []string{"pipeline_1"}, set.Names("pipelines"))
	assert.ElementsMatch(t,
		[]ResourceKey{
			{Type: "jobs", Name: "job_1"},
			{Type: "pipelines", Name: "pipeline_1"},
		},
		set.ToArray(),
	)
	assert.Equal(t, 2, set.Size())
}
