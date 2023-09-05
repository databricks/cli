package config

import (
	"testing"

	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestVerifyUniqueResourceIdentifiers(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		Models: map[string]*resources.MlflowModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
		Experiments: map[string]*resources.MlflowExperiment{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo2.yml",
				},
			},
		},
	}
	_, err := r.VerifyUniqueResourceIdentifiers()
	assert.ErrorContains(t, err, "multiple resources named foo (job at foo.yml, mlflow_experiment at foo2.yml)")
}

func TestVerifySafeMerge(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		Models: map[string]*resources.MlflowModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
	}
	other := Resources{
		Pipelines: map[string]*resources.Pipeline{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo2.yml",
				},
			},
		},
	}
	err := r.VerifySafeMerge(&other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at foo.yml, pipeline at foo2.yml)")
}

func TestVerifySafeMergeForSameResourceType(t *testing.T) {
	r := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo.yml",
				},
			},
		},
		Models: map[string]*resources.MlflowModel{
			"bar": {
				Paths: paths.Paths{
					ConfigFilePath: "bar.yml",
				},
			},
		},
	}
	other := Resources{
		Jobs: map[string]*resources.Job{
			"foo": {
				Paths: paths.Paths{
					ConfigFilePath: "foo2.yml",
				},
			},
		},
	}
	err := r.VerifySafeMerge(&other)
	assert.ErrorContains(t, err, "multiple resources named foo (job at foo.yml, job at foo2.yml)")
}
