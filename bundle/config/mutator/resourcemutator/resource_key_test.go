package resourcemutator

import (
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"

	"github.com/databricks/cli/libs/dyn"
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
			key, err := getResourceKey(dyn.MustPathFromString(tc.path))
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
	pattern  dyn.Pattern
	root     dyn.Value
	expected []ResourceKey
}

func TestResourceKeySet_AddPattern(t *testing.T) {
	root := dyn.V(map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"jobs": dyn.V(map[string]dyn.Value{
				"job_1": dyn.V(map[string]dyn.Value{
					"name": dyn.V("job_1"),
				}),
				"job_2": dyn.V(map[string]dyn.Value{
					"name": dyn.V("job_2"),
				}),
			}),
		}),
	})

	testCases := []resourceKeySetAddTestCase{
		{
			name:    "one job pattern",
			pattern: dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key("job_1")),
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
			pattern: dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
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
